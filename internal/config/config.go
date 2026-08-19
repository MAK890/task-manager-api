// Package config loads application settings from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server Server
	MySQL  MySQL
	Redis  Redis
}

type Server struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type MySQL struct {
	User     string
	Password string
	Address  string
	Database string
}

type Redis struct {
	Address  string
	Password string
	Database int
	CacheTTL time.Duration
}

func Load() (Config, error) {
	serverReadTimeout, err := duration("SERVER_READ_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	serverWriteTimeout, err := duration("SERVER_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	serverIdleTimeout, err := duration("SERVER_IDLE_TIMEOUT", time.Minute)
	if err != nil {
		return Config{}, err
	}
	serverShutdownTimeout, err := duration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cacheTTL, err := duration("REDIS_CACHE_TTL", time.Minute)
	if err != nil {
		return Config{}, err
	}
	redisDatabase, err := integer("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Server: Server{
			Address:         value("SERVER_ADDRESS", ":8080"),
			ReadTimeout:     serverReadTimeout,
			WriteTimeout:    serverWriteTimeout,
			IdleTimeout:     serverIdleTimeout,
			ShutdownTimeout: serverShutdownTimeout,
		},
		MySQL: MySQL{
			User:     value("MYSQL_USER", "task_api"),
			Password: os.Getenv("MYSQL_PASSWORD"),
			Address:  value("MYSQL_ADDRESS", "127.0.0.1:3306"),
			Database: value("MYSQL_DATABASE", "task_manager"),
		},
		Redis: Redis{
			Address:  value("REDIS_ADDRESS", "127.0.0.1:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			Database: redisDatabase,
			CacheTTL: cacheTTL,
		},
	}, nil
}

func value(name, fallback string) string {
	if configured := os.Getenv(name); configured != "" {
		return configured
	}
	return fallback
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	configured := os.Getenv(name)
	if configured == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(configured)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}
	return parsed, nil
}

func integer(name string, fallback int) (int, error) {
	configured := os.Getenv(name)
	if configured == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(configured)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("%s cannot be negative", name)
	}
	return parsed, nil
}
