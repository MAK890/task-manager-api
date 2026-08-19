// Package cache provides the Redis-backed application cache.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MAK890/task-manager-api/internal/config"
	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedis creates the Redis client and confirms that the server is reachable.
func NewRedis(ctx context.Context, cfg config.Redis) (*Redis, error) {
	// Ae janab Redis connection setup hega.
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.Database,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}

	return &Redis{client: client, ttl: cfg.CacheTTL}, nil
}

func (r *Redis) Close() error {
	return r.client.Close()
}

// Get decodes cached JSON into destination. A false hit is a normal cache miss.
func (r *Redis) Get(ctx context.Context, key string, destination any) (bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read key %q: %w", key, err)
	}

	if err := json.Unmarshal(data, destination); err != nil {
		// Invalid cache data nu remove kar ke MySQL ton dubara load karange.
		if deleteErr := r.client.Del(ctx, key).Err(); deleteErr != nil {
			return false, fmt.Errorf("decode key %q: %v; remove invalid value: %w", key, err, deleteErr)
		}
		return false, fmt.Errorf("decode key %q: %w", key, err)
	}

	return true, nil
}

// Set JSON-encodes a value and stores it with the configured expiration.
func (r *Redis) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode key %q: %w", key, err)
	}

	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("write key %q: %w", key, err)
	}
	return nil
}

func (r *Redis) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("delete keys %v: %w", keys, err)
	}
	return nil
}
