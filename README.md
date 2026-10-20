# 智能招聘系统

基于 AI Agent 能力的智能招聘系统，实现 HR 管理端 + 候选人用户端双前端系统，采用 Web + Logic 双服务 gRPC 微服务架构。

## 技术栈

### 后端
- Go + Gin（HTTP 网关）
- gRPC（服务间通信）
- GORM + MySQL（数据持久化）
- JWT（双角色认证鉴权）
- 阿里云 OSS（私有简历存储）
- Eino + DeepSeek（AI 问答）

### 前端
- React + TypeScript
- Vite
- React Router
- Axios

## 系统架构

```
HR 前端 (:3001)     候选人前端 (:3002)
        \               /
         \             /
          ↓  HTTP    ↓
    Web Gin 网关 (:8080)
    (路由/参数校验/JWT/CORS)
            ↓ gRPC
    Logic 核心服务 (:50051)
    (业务逻辑/MySQL/OSS/AI)
            ↓
   MySQL + 阿里云OSS + DeepSeek
```

## 环境要求

- Go 1.21+
- Node.js 18+
- MySQL 8.0+
- protoc（Protocol Buffers 编译器）
- protoc-gen-go / protoc-gen-go-grpc

## 配置说明

### MySQL 配置

1. 创建数据库并初始化表结构：

```bash
mysql -u root -p < logic-grpc-service/sql/init.sql
```

### 环境变量配置

在项目根目录复制示例并填写真实配置：

```bash
cp .env.example .env
```

MySQL/Redis 密码、OSS AccessKey、AI API Key 和 JWT 密钥均填入 `.env`。两个 Go 服务启动时自动读取根目录 `.env`，操作系统环境变量优先；原有 `config.yaml` 可保留非敏感配置，也可以省略。`.env` 不提交，`.env.example` 只保留占位值。

Docker Compose 也读取根目录 `.env`；首次初始化 MySQL 时使用非 root 的 `MYSQL_USER`，分别填写 `MYSQL_PASSWORD` 和 `MYSQL_ROOT_PASSWORD`。容器内的 MySQL、Redis 和 gRPC 地址由 Compose 指向对应服务。

### OSS 配置

1. 登录阿里云控制台创建 OSS Bucket
2. Bucket 权限设置为**私有**（关闭匿名访问和公开读）
3. 将 Endpoint、AccessKey、BucketName 填入根目录 `.env` 的对应 OSS 变量

## 服务启动顺序

### 1. 启动 MySQL

确保 MySQL 服务已运行，执行初始化 SQL：

```bash
mysql -u root -p < logic-grpc-service/sql/init.sql
```

### 2. 启动 Logic gRPC 服务

```bash
cd logic-grpc-service
go run main.go
```

看到 `Logic gRPC service listening on :50051` 表示启动成功。

### 3. 启动 Web Gin 网关

```bash
cd web-gin-service
go run main.go
```

看到 `Web Gin service listening on :8080` 表示启动成功。

### 4. 启动 HR 管理端前端

```bash
cd hr-frontend
npm install
npm run dev
```

访问 http://localhost:3001

### 5. 启动候选人用户端前端

```bash
cd user-frontend
npm install
npm run dev
```

访问 http://localhost:3002

## Proto 文件重新生成（如有修改）

```bash
cd logic-grpc-service
protoc --proto_path=proto \
  --go_out=pb --go_opt=module=logic-grpc-service/pb \
  --go-grpc_out=pb --go-grpc_opt=module=logic-grpc-service/pb \
  proto/*.proto
```

## 功能说明

### HR 管理端
- HR 注册/登录（JWT 认证）
- 岗位管理（新增/编辑/下架，仅管理自己的岗位）
- 查看投递列表（候选人信息、教育经历、技能标签、简历链接）
- AI 智能问答（基于 MySQL 真实数据，支持投递统计、候选人筛选等）
- 聊天记录持久化（刷新页面自动恢复历史对话）

### 候选人用户端
- 游客浏览公开岗位（无需登录）
- 候选人注册/登录
- 结构化个人资料编辑（姓名/电话/学历/院校/经历/技能）
- 简历上传（仅 PDF/DOC/DOCX，后端校验文件后缀+文件头）
- 岗位投递（需完善资料+上传简历后方可投递）

### 权限体系
- HR 仅能管理自己创建的岗位
- 候选人投递前必须完善资料 + 上传简历
- 游客仅能浏览岗位，不能投递

### AI 能力
- 使用 Eino 框架集成 DeepSeek 大模型
- AI 查询 MySQL 真实业务数据（投递统计/岗位热度/候选人筛选）
- 聊天记录自动入库，刷新页面后恢复历史上下文

## 项目亮点

1. **严格微服务分层**：Web 层零业务逻辑，所有业务通过 gRPC 调用 Logic 服务
2. **双角色 JWT 认证**：HR 和候选人独立账号体系，统一 JWT 鉴权
3. **私有 OSS 文件安全**：简历存储于私有 Bucket，通过签名 URL 访问
4. **AI 数据联动**：AI 问答基于 MySQL 真实业务数据，非静态模拟
5. **完整投递链路**：注册 → 完善资料 → 上传简历 → 投递 → HR 查看

## 部署说明

AI 多轮会话升级需对旧数据库执行一次 `logic-grpc-service/sql/migrations/20261020_chat_conversations.sql`，并同步升级 Logic、Web 和 HR 前端；已迁移的数据库无需重复执行。

1. 确保 MySQL、OSS、DeepSeek API 可用
2. 复制 `.env.example` 为 `.env` 并填写配置
3. 按顺序启动 Logic → Web → 前端
4. `.env` 和本地 `config.yaml` 已加入 `.gitignore`，只提交脱敏的 `.env.example`

## 目录结构

```
final_homework/
├── hr-frontend/                 # HR 管理端前端
├── user-frontend/               # 候选人用户端前端
├── web-gin-service/             # Gin Web 网关服务
├── logic-grpc-service/          # Logic 核心 gRPC 服务
│   └── sql/init.sql             # 数据库初始化脚本
├── api.md                       # 接口文档
├── db.md                        # 数据库设计文档
└── README.md                    # 项目说明
```
