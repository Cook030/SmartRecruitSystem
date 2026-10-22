-- Run once on an existing database before starting the upgraded Logic service.
ALTER TABLE async_tasks
  ADD COLUMN trace_parent VARCHAR(128) NOT NULL DEFAULT '' AFTER params,
  ADD COLUMN trace_state VARCHAR(512) NOT NULL DEFAULT '' AFTER trace_parent,
  ADD INDEX idx_status_created (status, created_at);
