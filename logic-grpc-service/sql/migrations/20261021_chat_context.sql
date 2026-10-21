-- Existing databases: apply 20261020_chat_conversations.sql first if needed.
-- Stop Logic writers and run once against the configured database before upgrading Logic.
-- No history is deleted or pre-marked as summarized; old conversations backfill on use.
ALTER TABLE `chat_conversations`
  ADD COLUMN `context_summary` TEXT NULL AFTER `title`,
  ADD COLUMN `summary_until_message_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `context_summary`,
  ADD COLUMN `summary_version` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `summary_until_message_id`;

ALTER TABLE `chat_messages`
  ADD COLUMN `context_notes` TEXT NULL AFTER `content`;
