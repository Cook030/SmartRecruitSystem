-- ============================================================
-- 智能招聘系统 - 数据库初始化脚本
-- ============================================================

CREATE DATABASE IF NOT EXISTS `smart_recruit`
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_general_ci;

USE `smart_recruit`;

-- -----------------------------------------------------------
-- 1. HR 用户表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `hr_users` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username`   VARCHAR(64)     NOT NULL,
  `password`   VARCHAR(255)    NOT NULL,
  `name`       VARCHAR(64)     NOT NULL DEFAULT '',
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 2. 候选人用户表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `candidates` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username`   VARCHAR(64)     NOT NULL,
  `password`   VARCHAR(255)    NOT NULL,
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 3. 候选人结构化档案表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `candidate_profiles` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `candidate_id` BIGINT UNSIGNED NOT NULL,
  `name`         VARCHAR(64)     NOT NULL DEFAULT '',
  `phone`        VARCHAR(20)     NOT NULL DEFAULT '',
  `education`    VARCHAR(32)     NOT NULL DEFAULT '',
  `university`   VARCHAR(128)    NOT NULL DEFAULT '',
  `experience`   TEXT,
  `skills`       VARCHAR(500)    NOT NULL DEFAULT '',
  `is_complete`  TINYINT(1)      NOT NULL DEFAULT 0,
  `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_candidate_id` (`candidate_id`),
  CONSTRAINT `fk_profile_candidate` FOREIGN KEY (`candidate_id`) REFERENCES `candidates` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 4. 岗位表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `jobs` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id`       BIGINT UNSIGNED NOT NULL,
  `title`       VARCHAR(128)    NOT NULL,
  `description` TEXT,
  `requirement` TEXT,
  `salary`      VARCHAR(64)     NOT NULL DEFAULT '',
  `location`    VARCHAR(128)    NOT NULL DEFAULT '',
  `status`      TINYINT(1)      NOT NULL DEFAULT 1 COMMENT '1=上架 0=下架',
  `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_hr_id` (`hr_id`),
  KEY `idx_status` (`status`),
  CONSTRAINT `fk_job_hr` FOREIGN KEY (`hr_id`) REFERENCES `hr_users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 5. 岗位投递关系表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `applications` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_id`       BIGINT UNSIGNED NOT NULL,
  `candidate_id` BIGINT UNSIGNED NOT NULL,
  `status`       TINYINT         NOT NULL DEFAULT 0 COMMENT '0=待处理 1=已查看',
  `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_job_candidate` (`job_id`, `candidate_id`),
  KEY `idx_candidate_id` (`candidate_id`),
  CONSTRAINT `fk_app_job` FOREIGN KEY (`job_id`) REFERENCES `jobs` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_app_candidate` FOREIGN KEY (`candidate_id`) REFERENCES `candidates` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 6. 简历 OSS 信息表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `resumes` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `candidate_id` BIGINT UNSIGNED NOT NULL,
  `file_name`    VARCHAR(255)    NOT NULL,
  `oss_key`      VARCHAR(500)    NOT NULL,
  `file_size`    BIGINT          NOT NULL DEFAULT 0,
  `file_type`    VARCHAR(20)     NOT NULL,
  `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_candidate_id` (`candidate_id`),
  CONSTRAINT `fk_resume_candidate` FOREIGN KEY (`candidate_id`) REFERENCES `candidates` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 7. AI 聊天记录表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `chat_messages` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id`      BIGINT UNSIGNED NOT NULL,
  `role`       VARCHAR(20)     NOT NULL COMMENT 'user=HR提问, assistant=AI回复',
  `content`    TEXT            NOT NULL,
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_hr_id` (`hr_id`),
  CONSTRAINT `fk_chat_hr` FOREIGN KEY (`hr_id`) REFERENCES `hr_users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 8. 异步任务表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `async_tasks` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_type`      VARCHAR(64)     NOT NULL COMMENT 'resume_parse/ai_match/stats_refresh',
  `business_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `user_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `user_role`      VARCHAR(20)     NOT NULL DEFAULT '',
  `status`         VARCHAR(20)     NOT NULL DEFAULT 'pending' COMMENT 'pending/running/success/failed/retrying/cancelled',
  `params`         TEXT,
  `result`         TEXT,
  `error_message`  TEXT,
  `retry_count`    INT             NOT NULL DEFAULT 0,
  `max_retries`    INT             NOT NULL DEFAULT 3,
  `started_at`     DATETIME        NULL,
  `completed_at`   DATETIME        NULL,
  `created_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_task_type_status` (`task_type`, `status`),
  KEY `idx_user` (`user_id`, `user_role`),
  KEY `idx_business` (`task_type`, `business_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 9. AI 简历解析结果表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `resume_parse_results` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `resume_id`    BIGINT UNSIGNED NOT NULL,
  `candidate_id` BIGINT UNSIGNED NOT NULL,
  `education`    TEXT,
  `experience`   TEXT,
  `projects`     TEXT,
  `skills`       VARCHAR(500)    NOT NULL DEFAULT '',
  `raw_result`   TEXT,
  `created_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_resume_id` (`resume_id`),
  KEY `idx_candidate_id` (`candidate_id`),
  CONSTRAINT `fk_parse_resume` FOREIGN KEY (`resume_id`) REFERENCES `resumes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 10. AI 岗位匹配结果表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `ai_match_results` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_id`          BIGINT UNSIGNED NOT NULL,
  `candidate_id`    BIGINT UNSIGNED NOT NULL,
  `application_id`  BIGINT UNSIGNED NOT NULL,
  `match_score`     DOUBLE          NOT NULL DEFAULT 0,
  `match_reason`    TEXT,
  `strengths`       TEXT,
  `weaknesses`      TEXT,
  `interview_tips`  TEXT,
  `raw_result`      TEXT,
  `created_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_application` (`application_id`),
  KEY `idx_job_id` (`job_id`),
  KEY `idx_candidate_id` (`candidate_id`),
  CONSTRAINT `fk_match_application` FOREIGN KEY (`application_id`) REFERENCES `applications` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- -----------------------------------------------------------
-- 11. 数据看板预计算统计表
-- -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS `dashboard_stats` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `hr_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '0=全局统计',
  `stat_key`   VARCHAR(128)    NOT NULL,
  `stat_value` TEXT            NOT NULL,
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_hr_stat` (`hr_id`, `stat_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
