// Package repository contains persistent data access implementations.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MAK890/task-manager-api/internal/model"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task model.Task) (int64, error) {
	// ID te created_at automatically MySQL vich generate honge.
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO tasks (title, description, status, priority) VALUES (?, ?, ?, ?)`,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
	)
	if err != nil {
		return 0, fmt.Errorf("insert task: %w", err)
	}

	// Itho last inserted task ID retrieve kar rahe haan.
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("retrieve last inserted task ID: %w", err)
	}
	return id, nil
}

func (r *TaskRepository) List(ctx context.Context) ([]model.Task, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, title, description, status, priority, created_at FROM tasks`,
	)
	if err != nil {
		return nil, fmt.Errorf("query tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]model.Task, 0)
	// MySQL vichon retrieved data nu scan kar ke tasks slice ch append kar rahe haan.
	for rows.Next() {
		var task model.Task
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.Priority,
			&task.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}
	return tasks, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id string) (model.Task, error) {
	var task model.Task
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, title, description, status, priority, created_at FROM tasks WHERE id = ?`,
		id,
	).Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.Priority,
		&task.CreatedAt,
	)
	if err != nil {
		return model.Task{}, err
	}
	return task, nil
}

func (r *TaskRepository) Update(ctx context.Context, id string, task model.Task) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE tasks SET title = ?, description = ?, status = ?, priority = ? WHERE id = ?`,
		task.Title,
		task.Description,
		task.Status,
		task.Priority,
		id,
	)
	if err != nil {
		return false, fmt.Errorf("update task: %w", err)
	}

	// Affected rows 0 hon da matlab requested task nahi labhi.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("retrieve updated row count: %w", err)
	}
	return rowsAffected > 0, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete task: %w", err)
	}

	// Affected rows 0 hon da matlab requested task nahi labhi.
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("retrieve deleted row count: %w", err)
	}
	return rowsAffected > 0, nil
}
