# 智能招聘系统

基于 AI Agent 能力的智能招聘系统，实现 HR 管理端 + 候选人用户端双前端系统，采用 Web + Logic 双服务 gRPC 微服务架构。

## 技术栈

### 后端
- Go + Gin（HTTP 网关）
- gRPC（服务间通信）
- GORM + MySQL（数据持久化）、Redis（缓存与任务队列）
- JWT（双角色认证鉴权）
- 阿里云 OSS（私有简历存储）
- Eino ReAct + DeepSeek（只读招聘工具调用与 AI 问答）

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
    (业务逻辑/MySQL/Redis/OSS/AI)
            ↓
   MySQL + Redis + 阿里云OSS + DeepSeek
```

## 环境要求

- Go 1.26.1+（与两个服务的 go.mod 一致）
- Node.js 20.19+（20.x）、22.13+（22.x）或 24+（满足 Vite 8 和 ESLint 10 要求）
- MySQL 8.0+；Redis（Compose 使用 Redis 7，连接失败时服务可降级启动）
- protoc（Protocol Buffers 编译器）
- protoc-gen-go / protoc-gen-go-grpc

## 配置说明

### MySQL 配置

1. 首次安装时创建数据库并初始化表结构（脚本固定使用 smart_recruit，MYSQL_DATABASE 需与之对应）：

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
2. Bucket 权限设置为私有（关闭匿名访问和公开读）
3. 将 Endpoint、AccessKey、BucketName 填入根目录 `.env` 的对应 OSS 变量

## 服务启动顺序

### 1. 启动 MySQL 和 Redis

确保 MySQL 和 Redis 服务已运行；首次安装执行初始化 SQL，已有数据库使用部署说明中的迁移脚本。后续各服务命令均从项目根目录在独立终端执行：

```bash
mysql -u root -p < logic-grpc-service/sql/init.sql
```

### 2. 启动 Logic gRPC 服务

```bash
cd logic-grpc-service
go run main.go
```

默认看到 `Logic gRPC service listening on :50051` 表示开始监听；实际端口由 GRPC_PORT 配置。

### 3. 启动 Web Gin 网关

```bash
cd web-gin-service
go run main.go
```

默认输出 `Web Gin service listening on :8080`，实际监听结果以 Gin 启动日志为准，端口由 WEB_PORT 配置。

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

cd ../web-gin-service
protoc --proto_path=proto \
  --go_out=pb --go_opt=module=web-gin-service/pb \
  --go-grpc_out=pb --go-grpc_opt=module=web-gin-service/pb \
  proto/*.proto
```

## 功能说明

### HR 管理端
- HR 注册/登录（JWT 认证）
- 岗位管理（新增/编辑/下架，仅管理自己的岗位）
- 查看投递列表（候选人信息、教育经历、技能标签、简历链接）
- AI 智能问答（通过只读工具查询当前 HR 的岗位及相关候选人，支持投递统计、候选人筛选等）
- 独立会话与聊天记录持久化（支持创建、切换、删除会话和历史分页；刷新后默认打开最新创建的会话）

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
- Eino ReAct 按需调用只读工具查询 MySQL，并限制在当前 HR 的数据范围内（投递统计/岗位热度/候选人筛选）
- 成功问答成对入库，请求 ID 支持幂等重试；上下文由会话交接文档、近期完整问答和本轮实时工具结果组成

### 对话上下文管理

按 Select → Structure → Compact → Persist 管理上下文，逻辑集中在 Logic 服务的 `chatcontext` 包，数据库读取和模型调用分别由 Repository、AI 层负责，无新增依赖：

- Select：读取当前 HR 当前会话最近最多 40 条尚未总结的消息，按 UTF-8 字节预算保留最近完整问答对；为交接文档、系统规则和当前问题预留空间。
- Structure：系统规则 → 历史交接资料 → 近期问答 → 当前问题；工具结果在本轮 ReAct 执行中追加。交接资料不作为系统指令或授权，当前明确条件优先，当前业务事实必须重新查工具。
- Compact：移出窗口的问答与旧文档增量合并，必须包含“当前目标、已完成事项、未完成任务、关键决策、失败经验、安全停点”六个非空字段。文档最多 6 KiB，字段缺失或超限则拒绝更新。无依据写“未知”，不编造完成状态或权限。
- Persist：完整原始问答继续保留，交接文档、覆盖消息 ID、版本与本轮成功问答在同一事务保存；请求租约和版本校验防止过期生成覆盖新进度。工具失败的内部记录随助手消息保存，不向历史接口暴露，也不保存原始 SQL 错误。

旧会话从尚未总结的最早消息开始回填；每批最多 20 条、48 KiB，每次请求最多 4 批，压缩阶段共用 20 秒超时。遇到异常或超长单轮时不跳过该轮、不推进失败批次的游标，后续请求仍可重试。压缩失败或回填尚未完成时，使用已有交接文档和近期问答，并提示模型历史存在缺口、需要澄清。只有本轮回答成功入库，已成功压缩批次的进度才一起提交。

上下文窗口使用字节估算。模型调用另有每请求累计 token 额度（`AI_TOKEN_BUDGET`，默认 128000），历史压缩与 Agent 各轮共用：调用前按消息、工具定义的 UTF-8 JSON 字节及最大输出预留，返回后按 usage 的输入与输出总量结算，额度不足时停止后续模型调用。单次输出最多 4096 token；usage 缺失或调用失败时保留预留扣额。预留是保守估算，实际 usage 超出估算时只能阻止后续调用，不能撤销已发生的消耗。该限制按请求生效，不是账户余额或跨请求账单。原有工具单次/累计结果限制继续生效，未实现执行过程中的动态摘要。结构校验保证六项存在，内容保真的效果仍需用真实模型与长对话案例验证。

验证命令：在 `logic-grpc-service` 下执行 `go test ./...`。测试使用模拟模型接口和业务 Reader，不调用真实 AI 或数据库。

## 项目亮点

1. 服务分层：Web 层处理路由、参数与文件校验、JWT 和响应映射，核心业务通过 gRPC 调用 Logic 服务
2. 双角色 JWT 认证：HR 和候选人独立账号体系，统一 JWT 鉴权
3. 私有 OSS 文件安全：简历存储于私有 Bucket，通过签名 URL 访问
4. AI 数据联动：AI 问答基于 MySQL 真实业务数据，非静态模拟
5. 完整投递链路：注册 → 完善资料 → 上传简历 → 投递 → HR 查看

## 部署说明

AI 多轮会话升级需对旧数据库执行一次 `logic-grpc-service/sql/migrations/20261020_chat_conversations.sql`，并同步升级 Logic、Web 和 HR 前端；已迁移的数据库无需重复执行。

上下文交接文档升级：已有会话表的数据库，在停止 Logic 写入后，对配置的数据库执行一次 `logic-grpc-service/sql/migrations/20261021_chat_context.sql`，然后启动新版 Logic。首次安装直接执行最新 `init.sql`，无需再执行上述迁移。本次未修改 HTTP/gRPC 协议和前端。

1. 确保 MySQL、Redis、OSS、DeepSeek API 可用
2. 复制 `.env.example` 为 `.env` 并填写配置
3. 按顺序启动 Logic → Web → 前端
4. `.env` 和本地 `config.yaml` 已加入 `.gitignore`，只提交脱敏的 `.env.example`

## 目录结构

```
SmartRecruitSystem/
├── hr-frontend/                 # HR 管理端前端
├── user-frontend/               # 候选人用户端前端
├── web-gin-service/             # Gin Web 网关服务
├── logic-grpc-service/          # Logic 核心 gRPC 服务
│   └── sql/init.sql             # 数据库初始化脚本
├── api.md                       # 接口文档
└── README.md                    # 项目说明
```

## 可观测性

Web 和 Logic 使用 OpenTelemetry 采集指标与链路，Prometheus 抓取独立管理端口，链路通过 OTLP/gRPC 发送至 Collector → Jaeger。Grafana 自动配置数据源和 Recruitment Observability 面板。未设置 OTLP 地址时，只提供指标和本地 Trace ID，不进行链路导出。

已有数据库升级前，停止 Logic 写入并执行一次 `logic-grpc-service/sql/migrations/20261022_observability.sql`；首次安装使用最新 `init.sql`，无需重复执行迁移。新增任务追踪字段只存 W3C `traceparent` / `tracestate`，不改变 HTTP/gRPC 协议，也不保存对话或简历正文。任务创建时保存上下文，Redis 消费和 MySQL 轮询都从任务记录恢复。

填写 `.env` 中原有业务配置及 `GRAFANA_ADMIN_PASSWORD`，启动完整 Docker 栈：

```sh
docker compose -f docker-compose.yml -f docker-compose.observability.yml up -d --build
```

- Grafana：<http://localhost:3000>，默认用户名 `admin`，密码使用 `.env` 配置。
- Prometheus：<http://localhost:9091>，Targets 查看两个服务，Alerts 查看规则状态。
- Jaeger：<http://localhost:16686>，服务名 `recruitment-web` / `recruitment-logic`。
- 指标：Web <http://localhost:9464/metrics>，Logic <http://localhost:9465/metrics>。

管理界面和指标端口默认只映射到本机。容器内的 OTLP 地址和指标监听地址由 Compose 覆盖；本地运行 Go 服务时，指标默认绑定 `127.0.0.1:9464` / `127.0.0.1:9465`，在进程环境或 `.env` 设置 `OTEL_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4317` 可接本地 Collector。若监控的是宿主机进程，需要将 Prometheus Targets 改为 `host.docker.internal:9464/9465`，并将 Go 管理端口绑定到 Docker 可访问的地址。示例全量采样，生产可降低 `OTEL_TRACES_SAMPLER_ARG`；采样只影响链路，不影响指标。

| 指标 | 维度与含义 |
|---|---|
| `recruitment_operations_total` / `recruitment_operation_duration_seconds` | 通过 `operation` 区分 HTTP、gRPC、任务执行、模型、工具、上下文压缩、数据库、Redis、OSS；通过 `outcome` 区分结果。耗时为 Histogram，支持 P95/P99。 |
| `recruitment_operations_inflight` | 各操作当前执行数，HTTP 带路由模板和方法。 |
| `recruitment_task_active` / `recruitment_task_oldest_age_seconds` | MySQL 中 pending/retrying/running 的数量和最老创建时间距今秒数。后台每 15 秒刷新，采集回调不查询数据库。 |
| `recruitment_task_stalled` | running 且开始执行超过 5 分钟的任务数。 |
| `recruitment_task_snapshot_up` / `recruitment_task_snapshot_timestamp_seconds` | 数据库快照采集是否成功、最近成功时间；失败时不继续报告旧积压值。 |
| `recruitment_task_redis_queue_length` / `recruitment_task_redis_up` | Redis 队列长度及最近一次探测结果，不能替代 MySQL 积压。 |
| `recruitment_task_latency_seconds` | `phase=first_start` 为首次开始距离创建的等待时间，`end_to_end` 为终态落库距离创建的总时间。 |
| `recruitment_ai_tokens_total` | provider 实际 usage，`kind=input/output/total`；缺失 usage 单独计事件，不把预算预留当实际消耗。汇总费用时只选 input/output 并乘以对应模型价格，避免与 total 重复相加。 |
| `recruitment_ai_request_model_attempts` / `recruitment_agent_tool_calls` | 每次 AI 请求的模型尝试次数（包含压缩）与每次 Agent 运行的工具调用次数，均为 Histogram。 |
| `recruitment_events_total` | `event` 包括任务重试/最终失败、入队和状态更新失败、Token/步数预算阻断、熔断拒绝、usage 缺失、AI JSON/分数异常、上下文降级和聊天幂等重放。 |
| `recruitment_db_connections` / `recruitment_db_pool_waits_total` / `recruitment_db_pool_wait_duration_seconds_total` | 连接池状态、累计等待次数和时间。另有标准 `go_*` / `process_*` 指标。 |

任务数据库快照是全局值，多副本按类型/状态取 `max`；操作 Counter 是进程内值，跨副本求和。任务执行、重试及终态事件按执行尝试计数，现有双 Worker 的调度和去重机制未重构，不能将事件 Counter 当成数据库中的唯一任务数。无效 AI JSON 会被计事件，沿用原有结果保存行为；观测到模型请求成功不代表结构化业务结果有效。

HTTP 响应附带 `X-Trace-ID`；请求、任务和工具的结构化 `slog` 日志附带 `trace_id/span_id`。SQL 文本、参数、Redis key/value、OSS 对象名、提示词和模型回复不进入新增指标或 Span；任务 ID 仅作为 Span 属性，指标只使用有界标签。Grafana 的 Prometheus 数据源配置了 exemplar → Jaeger 跳转。日志仍输出到标准输出，未部署日志存储服务。

示例告警覆盖服务采集失败、HTTP 5xx、任务等待/卡住、快照失效、Redis 不可用、AI 失败率及重试耗尽。阈值需要按业务时限调整；规则在 Prometheus 中展示，外部通知渠道需另行接入 Alertmanager。Prometheus 数据持久化 15 天，Jaeger 使用内存存储，重启会丢失链路。

验证：分别在两个 Go 服务目录执行 `go test ./...`。新增测试覆盖进程内 HTTP → gRPC 上下文传播、路由标签归一化、panic 状态码、异步上下文恢复、GORM dry-run 埋点、日志关联、指标导出以及 Token/预算计数，不依赖真实数据库、模型或 Collector。
