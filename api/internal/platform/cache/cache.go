// Package cache wraps the Redis client used for short lived data such as rate
// limit counters and cached responses.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Connect creates a Redis client and verifies connectivity with PING.
func Connect(ctx context.Context, url string) (*redis.Client, error) {
	options, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	client := redis.NewClient(options)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

// SetJSON stores value under key. A zero TTL keeps the key forever.
func SetJSON(ctx context.Context, client redis.Cmdable, key string, value any, ttl time.Duration) error {
	if client == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode cache value: %w", err)
	}
	if err := client.Set(ctx, key, encoded, ttl).Err(); err != nil {
		return fmt.Errorf("store cache value: %w", err)
	}
	return nil
}

// GetJSON loads the value stored under key. The boolean reports a cache hit.
func GetJSON(ctx context.Context, client redis.Cmdable, key string, dest any) (bool, error) {
	if client == nil {
		return false, nil
	}
	raw, err := client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load cache value: %w", err)
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, fmt.Errorf("decode cache value: %w", err)
	}
	return true, nil
}

// Delete removes the given keys.
func Delete(ctx context.Context, client redis.Cmdable, keys ...string) error {
	if client == nil || len(keys) == 0 {
		return nil
	}
	if err := client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("delete cache keys: %w", err)
	}
	return nil
}
