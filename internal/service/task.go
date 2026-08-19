// Package service contains task business rules and cache orchestration.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/MAK890/task-manager-api/internal/model"
)

const allTasksCacheKey = "tasks:all"

var (
	ErrTitleRequired   = errors.New("title is required")
	ErrInvalidPriority = errors.New("priority must be low, medium, or high")
	ErrTaskNotFound    = errors.New("task not found")
)

type Repository interface {
	Create(ctx context.Context, task model.Task) (int64, error)
	List(ctx context.Context) ([]model.Task, error)
	GetByID(ctx context.Context, id string) (model.Task, error)
	Update(ctx context.Context, id string, task model.Task) (bool, error)
	Delete(ctx context.Context, id string) (bool, error)
}

type Cache interface {
	Get(ctx context.Context, key string, destination any) (bool, error)
	Set(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, keys ...string) error
}

type TaskService struct {
	repository Repository
	cache      Cache
	logger     *log.Logger
}

func NewTaskService(repository Repository, cache Cache, logger *log.Logger) *TaskService {
	return &TaskService{repository: repository, cache: cache, logger: logger}
}

func taskCacheKey(id string) string {
	return "task:" + id
}

func validate(task model.Task) error {
	if task.Title == "" {
		return ErrTitleRequired
	}
	if task.Priority != "low" && task.Priority != "medium" && task.Priority != "high" {
		return ErrInvalidPriority
	}
	return nil
}

func (s *TaskService) Create(ctx context.Context, task model.Task) (int64, error) {
	s.logger.Printf("Task creation started: priority=%s", task.Priority)
	if err := validate(task); err != nil {
		s.logger.Printf("Task creation validation failed: error=%v", err)
		return 0, err
	}

	// Har navi task di status pending honi chahidi hegi.
	task.Status = "pending"
	id, err := s.repository.Create(ctx, task)
	if err != nil {
		s.logger.Printf("MySQL task creation failed: error=%v", err)
		return 0, fmt.Errorf("create task: %w", err)
	}
	s.logger.Printf("Task created in MySQL: id=%d", id)

	// Cached list hun outdated ho gayi hegi.
	s.invalidateCache(ctx, allTasksCacheKey)
	s.logger.Printf("Task creation completed: id=%d", id)
	return id, nil
}

func (s *TaskService) List(ctx context.Context) ([]model.Task, error) {
	s.logger.Println("Task list retrieval started")
	tasks := make([]model.Task, 0)
	// Pehlan Redis check karange; miss ya cache error te MySQL source of truth hai.
	if hit, err := s.cache.Get(ctx, allTasksCacheKey, &tasks); err != nil {
		s.logger.Printf("Redis cache read failed: key=%s error=%v", allTasksCacheKey, err)
	} else if hit {
		s.logger.Printf("Redis cache hit: key=%s task_count=%d", allTasksCacheKey, len(tasks))
		s.logger.Printf("Task list retrieval completed from Redis: task_count=%d", len(tasks))
		return tasks, nil
	} else {
		s.logger.Printf("Redis cache miss: key=%s", allTasksCacheKey)
	}

	s.logger.Println("Loading task list from MySQL")
	tasks, err := s.repository.List(ctx)
	if err != nil {
		s.logger.Printf("MySQL task list retrieval failed: error=%v", err)
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	s.logger.Printf("Task list loaded from MySQL: task_count=%d", len(tasks))
	if err := s.cache.Set(ctx, allTasksCacheKey, tasks); err != nil {
		s.logger.Printf("Redis cache write failed: key=%s error=%v", allTasksCacheKey, err)
	} else {
		s.logger.Printf("Redis cache populated: key=%s task_count=%d", allTasksCacheKey, len(tasks))
	}
	s.logger.Printf("Task list retrieval completed from MySQL: task_count=%d", len(tasks))
	return tasks, nil
}

func (s *TaskService) GetByID(ctx context.Context, id string) (model.Task, error) {
	key := taskCacheKey(id)
	s.logger.Printf("Task retrieval started: id=%s cache_key=%s", id, key)
	var task model.Task
	if hit, err := s.cache.Get(ctx, key, &task); err != nil {
		s.logger.Printf("Redis cache read failed: key=%s error=%v", key, err)
	} else if hit {
		s.logger.Printf("Redis cache hit: key=%s", key)
		s.logger.Printf("Task retrieval completed from Redis: id=%s", id)
		return task, nil
	} else {
		s.logger.Printf("Redis cache miss: key=%s", key)
	}

	s.logger.Printf("Loading task from MySQL: id=%s", id)
	task, err := s.repository.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		s.logger.Printf("Task not found in MySQL: id=%s", id)
		return model.Task{}, ErrTaskNotFound
	}
	if err != nil {
		s.logger.Printf("MySQL task retrieval failed: id=%s error=%v", id, err)
		return model.Task{}, fmt.Errorf("get task: %w", err)
	}
	s.logger.Printf("Task loaded from MySQL: id=%s", id)
	if err := s.cache.Set(ctx, key, task); err != nil {
		s.logger.Printf("Redis cache write failed: key=%s error=%v", key, err)
	} else {
		s.logger.Printf("Redis cache populated: key=%s", key)
	}
	s.logger.Printf("Task retrieval completed from MySQL: id=%s", id)
	return task, nil
}

func (s *TaskService) Update(ctx context.Context, id string, task model.Task) error {
	s.logger.Printf("Task update started: id=%s priority=%s", id, task.Priority)
	if err := validate(task); err != nil {
		s.logger.Printf("Task update validation failed: id=%s error=%v", id, err)
		return err
	}

	// Existing behavior mutabik update te status pending set hunda hai.
	task.Status = "pending"
	updated, err := s.repository.Update(ctx, id, task)
	if err != nil {
		s.logger.Printf("MySQL task update failed: id=%s error=%v", id, err)
		return fmt.Errorf("update task: %w", err)
	}
	if !updated {
		s.logger.Printf("Task update affected no rows: id=%s", id)
		return ErrTaskNotFound
	}
	s.logger.Printf("Task updated in MySQL: id=%s", id)

	s.invalidateCache(ctx, allTasksCacheKey, taskCacheKey(id))
	s.logger.Printf("Task update completed: id=%s", id)
	return nil
}

func (s *TaskService) Delete(ctx context.Context, id string) error {
	s.logger.Printf("Task deletion started: id=%s", id)
	deleted, err := s.repository.Delete(ctx, id)
	if err != nil {
		s.logger.Printf("MySQL task deletion failed: id=%s error=%v", id, err)
		return fmt.Errorf("delete task: %w", err)
	}
	if !deleted {
		s.logger.Printf("Task deletion found no row: id=%s", id)
		return ErrTaskNotFound
	}
	s.logger.Printf("Task deleted from MySQL: id=%s", id)

	s.invalidateCache(ctx, allTasksCacheKey, taskCacheKey(id))
	s.logger.Printf("Task deletion completed: id=%s", id)
	return nil
}

func (s *TaskService) invalidateCache(ctx context.Context, keys ...string) {
	if err := s.cache.Delete(ctx, keys...); err != nil {
		s.logger.Printf("Redis cache invalidation failed: keys=%v error=%v", keys, err)
		return
	}
	s.logger.Printf("Redis cache invalidated: keys=%v", keys)
}
