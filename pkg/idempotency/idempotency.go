// Package idempotency provides a Redis-backed dedup helper for message
// consumers.
//
// Usage pattern (post-mark, recommended):
//
//	seen, _ := cache.Seen(ctx, key)
//	if seen { msg.Ack(); return }                  // fast skip
//	if err := handler(ctx, msg); err != nil { ... } // run business logic
//	_ = cache.Mark(ctx, key)                       // record success
//	msg.Ack()
//
// Why post-mark instead of "Reserve before dispatch": if the handler
// crashes mid-flight, a pre-Reserved key would cause the redelivered
// message to be skipped — work lost. Post-mark only records success, so
// failures get retried. Handlers must be idempotent (e.g. via DB unique
// constraints) since redelivery may run them again.
//
// Reserve is kept for callers that genuinely want pre-claim semantics
// (e.g. HTTP idempotency-key header where the response body is cached).
package idempotency

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
	ttl time.Duration
}

func New(rdb *redis.Client, ttl time.Duration) *Cache {
	return &Cache{rdb: rdb, ttl: ttl}
}

// Seen reports whether key has been previously marked.
func (c *Cache) Seen(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, "idem:"+key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Mark records that key has been processed. Idempotent.
func (c *Cache) Mark(ctx context.Context, key string) error {
	return c.rdb.Set(ctx, "idem:"+key, "1", c.ttl).Err()
}

// Reserve atomically claims a key (SetNX). Returns true on first claim,
// false on duplicate. Use only when you need pre-claim semantics.
func (c *Cache) Reserve(ctx context.Context, key string) (bool, error) {
	ok, err := c.rdb.SetNX(ctx, "idem:"+key, "1", c.ttl).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	return ok, err
}
