# 智能招聘系统 - 数据库设计文档

## 数据库信息

- 数据库类型：MySQL 8.0+
- 数据库名：`smart_recruit`
- 字符集：`utf8mb4`
- 排序规则：`utf8mb4_general_ci`

---

## 表总览

| 序号 | 表名 | 说明 |
|------|------|------|
| 1 | `hr_users` | HR 用户表 |
| 2 | `candidates` | 候选人用户表 |
| 3 | `candidate_profiles` | 候选人结构化档案表 |
| 4 | `jobs` | 岗位表 |
| 5 | `applications` | 岗位投递关系表 |
| 6 | `resumes` | 简历 OSS 信息表 |
| 7 | `chat_messages` | AI 聊天记录表 |

---

## 1. hr_users - HR 用户表

存储 HR 管理员账号信息。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | HR ID |
| username | VARCHAR(64) | UNIQUE, NOT NULL | 登录用户名 |
| password | VARCHAR(255) | NOT NULL | bcrypt 加密后的密码 |
| name | VARCHAR(64) | NOT NULL, DEFAULT '' | HR 姓名 |
| created_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

索引：
- PRIMARY KEY (`id`)
- UNIQUE KEY `uk_username` (`username`)

---

## 2. candidates - 候选人用户表

存储候选人登录账号信息。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 候选人 ID |
| username | VARCHAR(64) | UNIQUE, NOT NULL | 登录用户名 |
| password | VARCHAR(255) | NOT NULL | bcrypt 加密后的密码 |
| created_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

索引：
- PRIMARY KEY (`id`)
- UNIQUE KEY `uk_username` (`username`)

---

## 3. candidate_profiles - 候选人结构化档案表

存储候选人的详细个人资料，与 candidates 表一对一关联。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 档案 ID |
| candidate_id | BIGINT UNSIGNED | UNIQUE, FK → candidates(id) | 候选人 ID |
| name | VARCHAR(64) | NOT NULL, DEFAULT '' | 姓名 |
| phone | VARCHAR(20) | NOT NULL, DEFAULT '' | 联系电话 |
| education | VARCHAR(32) | NOT NULL, DEFAULT '' | 最高学历 |
| university | VARCHAR(128) | NOT NULL, DEFAULT '' | 毕业院校 |
| experience | TEXT | | 工作/项目经历，JSON 格式 |
| skills | VARCHAR(500) | NOT NULL, DEFAULT '' | 技能标签，逗号分隔 |
| is_complete | TINYINT(1) | NOT NULL, DEFAULT 0 | 是否已完善资料：0=未完善, 1=已完善 |
| created_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

索引：
- PRIMARY KEY (`id`)
- UNIQUE KEY `uk_candidate_id` (`candidate_id`)

is_complete 判定规则：name、phone、education、university、experience、skills 均不为空时为 1。

---

## 4. jobs - 岗位表

存储 HR 发布的岗位信息。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 岗位 ID |
| hr_id | BIGINT UNSIGNED | FK → hr_users(id), NOT NULL | 创建者 HR ID |
| title | VARCHAR(128) | NOT NULL | 岗位名称 |
| description | TEXT | | 岗位描述 |
| requirement | TEXT | | 岗位要求 |
| salary | VARCHAR(64) | NOT NULL, DEFAULT '' | 薪资范围 |
| location | VARCHAR(128) | NOT NULL, DEFAULT '' | 工作地点 |
| status | TINYINT(1) | NOT NULL, DEFAULT 1 | 岗位状态：1=上架, 0=下架 |
| created_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

索引：
- PRIMARY KEY (`id`)
- KEY `idx_hr_id` (`hr_id`)
- KEY `idx_status` (`status`)

---

## 5. applications - 岗位投递关系表

记录候选人对岗位的投递关系。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 投递 ID |
| job_id | BIGINT UNSIGNED | FK → jobs(id), NOT NULL | 岗位 ID |
| candidate_id | BIGINT UNSIGNED | FK → candidates(id), NOT NULL | 候选人 ID |
| status | TINYINT | NOT NULL, DEFAULT 0 | 投递状态：0=待处理, 1=已查看 |
| created_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

索引：
- PRIMARY KEY (`id`)
- UNIQUE KEY `uk_job_candidate` (`job_id`, `candidate_id`) — 同一候选人对同一岗位只能投递一次
- KEY `idx_candidate_id` (`candidate_id`)

---

## 6. resumes - 简历 OSS 信息表

记录候选人上传到私有 OSS 的简历信息。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 简历 ID |
| candidate_id | BIGINT UNSIGNED | FK → candidates(id), NOT NULL | 候选人 ID |
| file_name | VARCHAR(255) | NOT NULL | 原始文件名 |
| oss_key | VARCHAR(500) | NOT NULL | OSS 对象 Key |
| file_size | BIGINT | NOT NULL, DEFAULT 0 | 文件大小（字节） |
| file_type | VARCHAR(20) | NOT NULL | 文件类型：pdf/doc/docx |
| created_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

索引：
- PRIMARY KEY (`id`)
- KEY `idx_candidate_id` (`candidate_id`)

---

## 7. chat_messages - AI 聊天记录表

存储 HR 与 AI 的对话记录，绑定 HR ID。

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | 消息 ID |
| hr_id | BIGINT UNSIGNED | FK → hr_users(id), NOT NULL | HR ID |
| role | VARCHAR(20) | NOT NULL | 角色：user=HR提问, assistant=AI回复 |
| content | TEXT | NOT NULL | 消息内容 |
| created_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | DATETIME | NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE | 更新时间 |

索引：
- PRIMARY KEY (`id`)
- KEY `idx_hr_id` (`hr_id`)

---

## 表关联关系

```
hr_users (1) ──→ (N) jobs              HR 创建岗位
hr_users (1) ──→ (N) chat_messages     HR 的聊天记录
candidates (1) ──→ (1) candidate_profiles   候选人档案
candidates (1) ──→ (N) resumes         候选人简历
candidates (1) ──→ (N) applications    候选人投递
jobs (1) ──→ (N) applications          岗位被投递
```
