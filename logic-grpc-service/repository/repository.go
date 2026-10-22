package repository

import (
	"context"
	"encoding/json"
	"log/slog"
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
			slog.WarnContext(ctx, "cache read failed", "error", err)
		}
	}

	data, err := queryFn()
	if err != nil {
		return err
	}

	if r.Redis.Available() {
		if setErr := r.Redis.Set(ctx, key, data, ttl); setErr != nil {
			slog.WarnContext(ctx, "cache write failed", "error", setErr)
		}
	}

	raw, _ := json.Marshal(data)
	return json.Unmarshal(raw, dest)
}

func (r *Repository) CacheDel(ctx context.Context, keys ...string) {
	if r.Redis.Available() {
		if err := r.Redis.Del(ctx, keys...); err != nil {
			slog.WarnContext(ctx, "cache delete failed", "error", err)
		}
	}
}

func (r *Repository) CacheDelByPrefix(ctx context.Context, prefix string) {
	if r.Redis.Available() {
		if err := r.Redis.DelByPrefix(ctx, prefix); err != nil {
			slog.WarnContext(ctx, "cache delete by prefix failed", "error", err)
		}
	}
}
