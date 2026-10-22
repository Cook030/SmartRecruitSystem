package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"logic-grpc-service/config"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	rdb *redis.Client
}

var GlobalClient *Client

func NewClient(cfg config.RedisConfig) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	rdb.AddHook(telemetryHook{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.WarnContext(ctx, "Redis connection failed; using MySQL polling without cache", "error", err)
		_ = rdb.Close()
		return &Client{rdb: nil}
	}

	slog.InfoContext(ctx, "Redis connected")
	return &Client{rdb: rdb}
}

func (c *Client) Available() bool {
	return c.rdb != nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if !c.Available() {
		return "", redis.Nil
	}
	return c.rdb.Get(ctx, key).Result()
}

func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if !c.Available() {
		return redis.Nil
	}
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !c.Available() {
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	default:
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	if !c.Available() {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Client) DelByPrefix(ctx context.Context, prefix string) error {
	if !c.Available() {
		return nil
	}
	iter := c.rdb.Scan(ctx, 0, prefix+"*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if len(keys) > 0 {
		return c.rdb.Del(ctx, keys...).Err()
	}
	return nil
}

func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	if !c.Available() {
		return nil
	}
	return c.rdb.LPush(ctx, key, values...).Err()
}

func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	if !c.Available() {
		return "", redis.Nil
	}
	return c.rdb.RPop(ctx, key).Result()
}

func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	if !c.Available() {
		return 0, nil
	}
	return c.rdb.LLen(ctx, key).Result()
}

func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	if !c.Available() {
		return nil
	}
	return c.rdb.HSet(ctx, key, values...).Err()
}

func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if !c.Available() {
		return nil, redis.Nil
	}
	return c.rdb.HGetAll(ctx, key).Result()
}

func (c *Client) BRPop(ctx context.Context, timeout time.Duration, key string) (string, error) {
	if !c.Available() {
		return "", redis.Nil
	}
	result, err := c.rdb.BRPop(ctx, timeout, key).Result()
	if err != nil {
		return "", err
	}
	if len(result) < 2 {
		return "", fmt.Errorf("unexpected brpop result")
	}
	return result[1], nil
}
