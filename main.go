// Package main sade compiler nu dasda hega ke application executable hai.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/MAK890/task-manager-api/internal/cache"
	"github.com/MAK890/task-manager-api/internal/config"
	"github.com/MAK890/task-manager-api/internal/database"
	"github.com/MAK890/task-manager-api/internal/httpapi"
	"github.com/MAK890/task-manager-api/internal/repository"
	"github.com/MAK890/task-manager-api/internal/server"
	"github.com/MAK890/task-manager-api/internal/service"
)

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.LUTC)
	logger.Println("Application startup initiated")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger); err != nil {
		logger.Fatalf("Application stopped with error: %v", err)
	}
	logger.Println("Application stopped cleanly")
}

// run application de dependencies setup karda; business logic apne packages ch hai.
func run(ctx context.Context, logger *log.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	logger.Printf(
		"Configuration loaded: server_address=%s mysql_address=%s mysql_database=%s mysql_user=%s redis_address=%s redis_db=%d cache_ttl=%s",
		cfg.Server.Address,
		cfg.MySQL.Address,
		cfg.MySQL.Database,
		cfg.MySQL.User,
		cfg.Redis.Address,
		cfg.Redis.Database,
		cfg.Redis.CacheTTL,
	)

	// MySQL connection application shutdown te close ho jega.
	logger.Printf("Connecting to MySQL: address=%s database=%s user=%s", cfg.MySQL.Address, cfg.MySQL.Database, cfg.MySQL.User)
	db, err := database.OpenMySQL(ctx, cfg.MySQL)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Printf("MySQL connection close failed: error=%v", err)
			return
		}
		logger.Println("MySQL connection closed")
	}()
	logger.Println("Connected to MySQL")

	// Exactly same lifecycle Redis client layi.
	logger.Printf("Connecting to Redis: address=%s database=%d", cfg.Redis.Address, cfg.Redis.Database)
	redisCache, err := cache.NewRedis(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer func() {
		if err := redisCache.Close(); err != nil {
			logger.Printf("Redis connection close failed: error=%v", err)
			return
		}
		logger.Println("Redis connection closed")
	}()
	logger.Println("Connected to Redis")

	logger.Println("Creating repository, service, HTTP handler, and server dependencies")
	taskRepository := repository.NewTaskRepository(db)
	taskService := service.NewTaskService(taskRepository, redisCache, logger)
	taskHandler := httpapi.NewHandler(taskService, logger)
	httpServer := server.New(cfg.Server, taskHandler.Routes(), logger)

	return httpServer.Run(ctx)
}
