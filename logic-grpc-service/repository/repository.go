package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rds "logic-grpc-service/redis"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Repository struct {
	DB    *gorm.DB
	Redis *rds.Client
}

func NewRepository(db *gorm.DB, rdb *rds.Client) *Repository {
	return &Repository{DB: db, Redis: rdb}
}

// WithContext returns a request-scoped handle; shared service state stays unchanged.
func (r *Repository) WithContext(ctx context.Context) *Repository {
	return &Repository{DB: r.DB.WithContext(ctx), Redis: r.Redis}
}

func (r *Repository) CacheGet(ctx context.Context, key string, dest interface{}, ttl time.Duration, queryFn func() (interface{}, error)) error {
	if r.Redis.Available() {
		err := r.Redis.GetJSON(ctx, key, dest)
		if err == nil {
			return nil
		}
		if err != redis.Nil {
			fmt.Printf("cache read error for key %s: %v\n", key, err)
		}
	}

	data, err := queryFn()
	if err != nil {
		return err
	}

	if r.Redis.Available() {
		if setErr := r.Redis.Set(ctx, key, data, ttl); setErr != nil {
			fmt.Printf("cache write error for key %s: %v\n", key, setErr)
		}
	}

	raw, _ := json.Marshal(data)
	return json.Unmarshal(raw, dest)
}

func (r *Repository) CacheDel(ctx context.Context, keys ...string) {
	if r.Redis.Available() {
		if err := r.Redis.Del(ctx, keys...); err != nil {
			fmt.Printf("cache delete error: %v\n", err)
		}
	}
}

func (r *Repository) CacheDelByPrefix(ctx context.Context, prefix string) {
	if r.Redis.Available() {
		if err := r.Redis.DelByPrefix(ctx, prefix); err != nil {
			fmt.Printf("cache delete by prefix error: %v\n", err)
		}
	}
}
