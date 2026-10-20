-- Existing databases only. Back up first, stop Logic/Web writers, then run once.
-- USE your configured database before executing this file. MySQL DDL commits.
CREATE TABLE `chat_conversations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id` BIGINT UNSIGNED NOT NULL,
  `title` VARCHAR(80) NOT NULL DEFAULT '新对话',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_chat_conversation_hr` (`hr_id`, `id`),
  CONSTRAINT `fk_chat_conversation_hr` FOREIGN KEY (`hr_id`) REFERENCES `hr_users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

CREATE TABLE `chat_requests` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id` BIGINT UNSIGNED NOT NULL,
  `conversation_id` BIGINT UNSIGNED NOT NULL,
  `request_id` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `content` TEXT NOT NULL,
  `reply` LONGTEXT NOT NULL,
  `status` VARCHAR(20) NOT NULL,
  `attempt` VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `lease_until` DATETIME(6) NOT NULL,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_chat_request` (`hr_id`, `request_id`),
  KEY `idx_chat_request_active` (`conversation_id`, `status`, `lease_until`),
  CONSTRAINT `fk_chat_request_conversation` FOREIGN KEY (`conversation_id`) REFERENCES `chat_conversations` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

ALTER TABLE `chat_messages` ADD COLUMN `conversation_id` BIGINT UNSIGNED NULL AFTER `hr_id`;

-- One legacy conversation per HR, preserving message IDs and ordering.
INSERT INTO `chat_conversations` (`hr_id`, `title`, `created_at`, `updated_at`)
SELECT `hr_id`, '历史对话', MIN(`created_at`), MAX(`updated_at`)
FROM `chat_messages` GROUP BY `hr_id`;

UPDATE `chat_messages` m JOIN `chat_conversations` c ON c.hr_id = m.hr_id
SET m.conversation_id = c.id, m.updated_at = m.updated_at;

ALTER TABLE `chat_messages`
  MODIFY COLUMN `conversation_id` BIGINT UNSIGNED NOT NULL,
  ADD KEY `idx_chat_conversation_message` (`conversation_id`, `id`),
  ADD CONSTRAINT `fk_chat_message_conversation` FOREIGN KEY (`conversation_id`) REFERENCES `chat_conversations` (`id`) ON DELETE CASCADE;
