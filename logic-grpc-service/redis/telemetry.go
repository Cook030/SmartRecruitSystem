package redis

import (
	"context"
	"errors"
	"net"

	"logic-grpc-service/telemetry"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
)

// Hooks observe command names only; cache keys and values are never exported.
type telemetryHook struct{}

func (telemetryHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (telemetryHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		name := cmd.Name()
		switch name {
		case "ping", "get", "set", "del", "scan", "lpush", "rpop", "llen", "hset", "hgetall", "brpop":
		default:
			name = "other"
		}
		ctx, end := telemetry.Start(ctx, "redis.command", attribute.String("command", name))
		err := next(ctx, cmd)
		outcome := telemetry.Outcome(err)
		if errors.Is(err, redis.Nil) {
			outcome = "miss"
		}
		var timeout net.Error
		if errors.As(err, &timeout) && timeout.Timeout() {
			outcome = "timeout"
		}
		end(outcome)
		return err
	}
}
func (telemetryHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		ctx, end := telemetry.Start(ctx, "redis.pipeline")
		err := next(ctx, cmds)
		end(telemetry.Outcome(err))
		return err
	}
}
