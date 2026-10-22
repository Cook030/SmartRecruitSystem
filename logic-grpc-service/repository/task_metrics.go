package repository

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type taskMetricRow struct {
	TaskType string
	Status   string
	Count    int64
	Oldest   time.Time
	Stalled  int64
}

// StartTaskMetrics polls outside scrape callbacks. Each replica reports the same
// database-wide state, so dashboards must use max, not sum, across replicas.
func (r *Repository) StartTaskMetrics(parent context.Context) (func(), error) {
	m := otel.Meter("recruitment")
	count, _ := m.Int64ObservableGauge("recruitment.task.active")
	oldest, _ := m.Float64ObservableGauge("recruitment.task.oldest.age", metric.WithUnit("s"))
	stalled, _ := m.Int64ObservableGauge("recruitment.task.stalled")
	queue, _ := m.Int64ObservableGauge("recruitment.task.redis.queue.length")
	up, _ := m.Int64ObservableGauge("recruitment.task.snapshot.up")
	redisUp, _ := m.Int64ObservableGauge("recruitment.task.redis.up")
	updated, _ := m.Float64ObservableGauge("recruitment.task.snapshot.timestamp", metric.WithUnit("s"))
	var mu sync.Mutex
	var rows []taskMetricRow
	var healthy, redisHealthy, length int64
	var timestamp float64
	ctx, cancel := context.WithCancel(parent)
	reg, err := m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		mu.Lock()
		defer mu.Unlock()
		o.ObserveInt64(up, healthy)
		o.ObserveInt64(redisUp, redisHealthy)
		o.ObserveFloat64(updated, timestamp)
		if redisHealthy == 1 {
			o.ObserveInt64(queue, length)
		}
		if healthy == 0 {
			return nil
		}
		for _, row := range rows {
			attrs := metric.WithAttributes(attribute.String("task_type", row.TaskType), attribute.String("status", row.Status))
			o.ObserveInt64(count, row.Count, attrs)
			age := 0.0
			if row.Count > 0 {
				age = max(0, time.Since(row.Oldest).Seconds())
			}
			o.ObserveFloat64(oldest, age, attrs)
			o.ObserveInt64(stalled, row.Stalled, attrs)
		}
		return nil
	}, count, oldest, stalled, queue, up, redisUp, updated)
	if err != nil {
		cancel()
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			poll, stop := context.WithTimeout(ctx, 3*time.Second)
			var result []taskMetricRow
			// Restrict to active tasks, keeping completed history out of this query.
			kind := "CASE WHEN task_type IN ('resume_parse','ai_match','stats_refresh','notification') THEN task_type ELSE 'other' END"
			err := r.DB.WithContext(poll).Table("async_tasks").
				Select(kind+" AS task_type, status, COUNT(*) AS count, MIN(created_at) AS oldest, SUM(CASE WHEN status = 'running' AND started_at < ? THEN 1 ELSE 0 END) AS stalled", time.Now().Add(-5*time.Minute)).
				Where("status IN ?", []string{"pending", "retrying", "running"}).Group(kind + ", status").Scan(&result).Error
			var queueLength int64
			var redisOK int64
			if r.Redis.Available() {
				if n, e := r.Redis.LLen(poll, "task:queue"); e == nil {
					queueLength, redisOK = n, 1
				}
			}
			stop()
			mu.Lock()
			healthy = 0
			if err == nil {
				healthy, timestamp = 1, float64(time.Now().Unix())
				// Emit zeros for empty combinations, so drained queues are visible.
				rows = nil
				for _, taskType := range []string{"resume_parse", "ai_match", "stats_refresh", "notification", "other"} {
					for _, state := range []string{"pending", "retrying", "running"} {
						row := taskMetricRow{TaskType: taskType, Status: state}
						for _, found := range result {
							if found.TaskType == taskType && found.Status == state {
								row = found
								break
							}
						}
						rows = append(rows, row)
					}
				}
			}
			length, redisHealthy = queueLength, redisOK
			mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return func() { cancel(); <-done; _ = reg.Unregister() }, nil
}
