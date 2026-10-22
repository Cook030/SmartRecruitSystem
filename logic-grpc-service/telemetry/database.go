package telemetry

import (
	"context"
	"database/sql"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"
)

// DatabasePlugin deliberately excludes SQL text, parameters and table names.
type DatabasePlugin struct{}

func (DatabasePlugin) Name() string { return "recruitment_telemetry" }
func (DatabasePlugin) Initialize(db *gorm.DB) error {
	c := db.Callback()
	return errors.Join(
		c.Create().Before("gorm:create").Register("telemetry:before", beforeDB("create")),
		c.Create().After("gorm:create").Register("telemetry:after", afterDB),
		c.Query().Before("gorm:query").Register("telemetry:before", beforeDB("query")),
		c.Query().After("gorm:query").Register("telemetry:after", afterDB),
		c.Update().Before("gorm:update").Register("telemetry:before", beforeDB("update")),
		c.Update().After("gorm:update").Register("telemetry:after", afterDB),
		c.Delete().Before("gorm:delete").Register("telemetry:before", beforeDB("delete")),
		c.Delete().After("gorm:delete").Register("telemetry:after", afterDB),
		c.Row().Before("gorm:row").Register("telemetry:before", beforeDB("row")),
		c.Row().After("gorm:row").Register("telemetry:after", afterDB),
		c.Raw().Before("gorm:raw").Register("telemetry:before", beforeDB("raw")),
		c.Raw().After("gorm:raw").Register("telemetry:after", afterDB),
	)
}

func RegisterDBPool(db *sql.DB) (metric.Registration, error) {
	connections, _ := meter.Int64ObservableGauge("recruitment.db.connections")
	waits, _ := meter.Int64ObservableCounter("recruitment.db.pool.waits")
	waitTime, _ := meter.Float64ObservableCounter("recruitment.db.pool.wait.duration", metric.WithUnit("s"))
	return meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		s := db.Stats()
		for state, n := range map[string]int{"in_use": s.InUse, "idle": s.Idle, "max_open": s.MaxOpenConnections} {
			o.ObserveInt64(connections, int64(n), metric.WithAttributes(attribute.String("state", state)))
		}
		o.ObserveInt64(waits, s.WaitCount)
		o.ObserveFloat64(waitTime, s.WaitDuration.Seconds())
		return nil
	}, connections, waits, waitTime)
}

func beforeDB(operation string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		parent := db.Statement.Context
		ctx, end := Start(parent, "db."+operation)
		db.Statement.Context = ctx
		db.InstanceSet("telemetry.end", func() {
			outcome := Outcome(db.Error)
			if errors.Is(db.Error, gorm.ErrRecordNotFound) {
				outcome = "not_found"
			}
			end(outcome)
			db.Statement.Context = parent
		})
	}
}
func afterDB(db *gorm.DB) {
	if value, ok := db.InstanceGet("telemetry.end"); ok {
		value.(func())()
	}
}
