# 智能招聘系统 AI Agent 集成拓展设计方案

## 文档说明

本文档基于当前智能招聘系统的已有工程能力进行扩展设计。现有系统采用 HR 管理端 + 候选人用户端双前端，后端采用 Go + Gin HTTP 网关、Logic gRPC 核心服务、GORM + MySQL、JWT 双角色鉴权、阿里云 OSS 私有简历存储，并已通过 Eino + DeepSeek 实现基于真实业务数据的 AI 问答能力。

本方案面向企业级 AI Agent 生态，详细设计系统如何接入 OpenClaw、Hermes 等第三方 AI Agent 平台，包括接入流程、交互协议、工具注册、任务编排和差异化适配等内容，为系统从传统业务平台升级为可被 Agent 调度的 AI Native 招聘平台提供设计依据。

---

# 1. AI Agent 集成总体流程

## 1.1 完整交互流程

1. **用户输入自然语言**：用户可以是 HR、候选人、招聘管理者或第三方 Agent 平台。
2. **请求进入统一入口**：浏览器端请求进入 Web Gin Gateway，第三方 Agent 请求进入 Agent Gateway，MCP Client 请求进入 MCP Server。
3. **身份认证与权限解析**：解析 JWT、OAuth2 Token、API Key 或 Agent Token，识别用户身份、角色、数据范围和可调用工具。
4. **上下文加载**：从 Redis 和 MySQL 加载短期会话、历史聊天记录、用户偏好、HR 管理范围和候选人档案状态。
5. **意图识别**：判断请求属于信息查询、内容生成、分析决策、业务执行或知识问答。
6. **任务规划与工具选择**：Agent Orchestrator 根据意图选择 Tool，并生成执行计划。
7. **工具调用**：Agent 通过 Tool Calling 调用招聘系统 API 或 gRPC 服务，获取岗位、候选人、简历和投递数据。
8. **RAG 检索增强**：从企业知识库、岗位知识库、简历库和面试题库中检索相关片段。
9. **模型推理**：LLM 基于 System Prompt、Role Prompt、Tool Result、RAG Context 和输出 Schema 生成回答。
10. **安全校验**：AI Safety Guardrail 检查越权访问、敏感信息泄露、Prompt Injection、幻觉风险和违规招聘建议。
11. **结果落库**：聊天内容写入 `chat_messages`，工具调用写入审计日志。
12. **返回结果**：返回自然语言说明，并附带结构化表格、候选人列表、推荐理由或可执行操作建议。

## 1.2 Agent 调用招聘系统 API 的流程

以第三方 Agent 调用"获取岗位推荐结果"为例：第三方 Agent 使用 OAuth2 Client Credentials 或 Agent Token 访问 Agent Gateway；Agent Gateway 校验调用方身份并读取授权 Tool 列表；调用方请求 `recommend_jobs_for_candidate` 工具；Tool Registry 校验权限；Agent Gateway 将工具调用转换为内部 gRPC 或 REST 请求；Logic Service 查询候选人档案、投递历史和上架岗位；RAG Service 检索岗位描述和候选人简历摘要；Match Scoring Service 计算匹配度；Agent Gateway 对结果脱敏和权限裁剪后返回结构化推荐结果。

## 1.3 执行型任务确认机制

创建面试、下架岗位、更新投递状态、发送候选人沟通消息等写操作必须引入二次确认。Agent 先生成拟执行计划和影响范围，系统展示关键参数，用户确认后才调用真实业务 API。执行结果必须写入审计日志；若失败，Agent 必须返回失败原因和可重试建议，不得伪造成功结果。

## 1.4 对话交互模式

系统支持四种交互模式：同步问答适用于简单查询，一次性返回完整回答；流式输出通过 SSE 或 WebSocket 逐步返回内容，适合长文本生成；多轮对话支持 Agent 主动提问和上下文继承，适合需求澄清和二次确认；异步长任务用于批量操作，返回任务 ID 供轮询或回调。对话状态包括空闲、意图识别中、工具调用中、模型推理中、等待确认、完成和异常，状态变更通过事件通知前端。多轮对话通过 session_id 关联上下文，短期上下文存缓存，长期上下文存数据库，每次请求自动加载相关上下文注入模型。

---

# 2. OpenClaw / Hermes 接入设计

## 2.1 接入流程时序

以第三方 Agent 平台接入本系统为例，完整接入流程如下：

**第一阶段：接入准备**
1. 第三方 Agent 平台向系统申请接入，提供平台标识、回调地址和所需能力范围
2. 系统审核接入申请，分配 Agent Token 和 API Key，配置权限范围和限流策略
3. 系统向 Agent 平台暴露 Tool Discovery 端点，返回可用工具列表和 Schema

**第二阶段：工具发现与注册**
4. Agent 平台调用 Tool Discovery 端点，获取系统暴露的工具描述
5. Agent 平台将工具注册到自身 Tool Registry，建立工具名称与系统端点的映射
6. Agent 平台配置工具调用策略，包括重试机制、超时处理和错误恢复

**第三阶段：运行时交互**
7. 用户向 Agent 平台发起自然语言请求
8. Agent 平台识别意图，选择需要调用的工具
9. Agent 平台携带 Agent Token 调用系统 Agent Gateway 的工具执行端点
10. Agent Gateway 校验 Token、权限和限流，转发请求到内部服务
11. 内部服务执行业务逻辑，返回结果
12. Agent Gateway 对结果脱敏和裁剪后返回给 Agent 平台
13. Agent 平台将工具结果注入模型上下文，生成最终回答返回用户

**第四阶段：监控与审计**
14. 系统记录工具调用的审计日志，包含调用方、参数摘要、耗时和结果状态
15. Agent 平台记录 Agent 决策日志，包含意图识别、工具选择和模型输出
16. 双方通过 Webhook 或回调机制同步异常事件和状态变更

## 2.2 OpenClaw 接入方案

### 2.2.1 接入定位

OpenClaw 可作为第三方 Agent Framework 接入本系统，用于构建 HR 招聘助手、候选人求职助手、招聘数据分析助手等智能体。系统需要将自身业务能力注册为 OpenClaw 可识别和调用的 Tool，使 OpenClaw Agent 能通过 Tool Calling 完成招聘相关任务。OpenClaw 不直接访问 MySQL 或 OSS，而通过 Agent Gateway 调用受控工具。

### 2.2.2 工具注册设计原则

工具注册是将系统业务能力暴露给 Agent 的核心机制。设计时应遵循以下原则：

- **语义明确**：每个工具名称和描述应清晰表达其能力边界，避免模糊或重叠
- **参数结构化**：入参和出参采用 JSON Schema 定义，避免自由文本拼接
- **权限绑定**：每个工具声明所需角色和数据范围，由 Agent Gateway 统一校验
- **风险分级**：只读工具标记为低风险，写操作工具标记为高风险并要求二次确认
- **脱敏策略**：工具返回结果根据调用方身份自动脱敏敏感字段
- **超时与限流**：每个工具定义调用超时时间和频率限制，防止滥用

### 2.2.3 API 暴露方式

OpenClaw 接入时，系统通过 Agent Gateway 暴露 Tool Discovery、Tool Invoke、Tool Schema、Agent Session、Callback 等端点。这样可以统一处理 Agent Token、API Key、OAuth2、签名校验、限流、审计、脱敏和 Prompt Injection 防护，同时隐藏内部服务结构。

### 2.2.4 Tool Calling 与 Function Calling

OpenClaw Agent 根据用户问题选择工具，根据 Schema 生成结构化参数；Agent Gateway 校验参数、身份、权限和限流；内部调用 Logic Service 执行业务逻辑；返回结果经脱敏、裁剪和格式化后注入模型上下文。Function Calling 设计必须避免让模型拼接 SQL，所有函数均应语义明确、参数结构化、输出稳定；写操作函数标记为高风险并要求确认；执行过程记录调用方、参数摘要、耗时和审计 ID。

### 2.2.5 Prompt 与上下文管理

OpenClaw 场景需要 System Prompt、Tool Prompt、Role Prompt、Output Prompt 和 Safety Prompt。System Prompt 约束 Agent 是企业招聘系统助手；Tool Prompt 说明工具用途和失败处理；Role Prompt 区分 HR、候选人和管理者权限；Output Prompt 约束结构化输出；Safety Prompt 禁止泄露敏感信息、绕过权限和输出歧视性建议。

上下文分为会话上下文、用户上下文、业务上下文和知识上下文。会话上下文存 Redis + MySQL，业务上下文来自 MySQL + OSS + Vector DB，知识上下文来自向量数据库。注入模型前必须进行 Token 控制和权限过滤。

## 2.3 Hermes 接入方案

### 2.3.1 Agent Workflow 定位

Hermes 更适合复杂任务流编排。相比 OpenClaw 偏工具调用，Hermes 可将招聘任务拆解为多个 Workflow Node，由多个 Agent 或工具节点协同完成，适用于批量简历解析、岗位候选人匹配、面试安排、面试结果汇总、招聘周报生成和候选人自动沟通。

### 2.3.2 Workflow 设计

| 节点 | 类型 | 输入 | 输出 |
|---|---|---|---|
| Intent Node | Agent | HR 自然语言需求 | 任务目标、岗位条件、筛选规则 |
| Job Query Node | Tool | 岗位名称或岗位 ID | 岗位详情 |
| Application Query Node | Tool | 岗位 ID | 投递候选人列表 |
| Resume Parse Node | Tool / Agent | 简历 OSS Key | 简历结构化信息 |
| RAG Retrieval Node | Tool | 岗位要求、候选人关键词 | 相关知识片段 |
| Match Analysis Node | Agent | 岗位、候选人、简历、知识片段 | 匹配评分和推荐理由 |
| Ranking Node | Tool | 匹配评分列表 | 排序后的候选人列表 |
| Report Node | Agent | 排序结果 | HR 可读分析报告 |
| Audit Node | System | 调用过程和结果 | 审计记录 |

### 2.3.3 任务流配置与多 Agent 协作

Hermes 任务流应支持节点定义、DAG 依赖关系、条件分支、人工确认节点和异常补偿节点。招聘 Agent 负责理解 HR 目标并拆解任务；简历分析 Agent 负责解析简历；岗位推荐 Agent 负责匹配推荐；面试 Agent 负责面试题和总结；数据分析 Agent 负责招聘指标；安全审查 Agent 负责敏感信息和合规检查。

在候选人筛选中，招聘 Agent 确定岗位和候选人范围，调用简历分析 Agent 处理 OSS 简历，简历分析 Agent 返回技能、项目、年限、学历和风险提示，招聘 Agent 再调用匹配分析节点并汇总输出。任务支持同步模式和异步模式，状态包括 `pending`、`running`、`waiting_confirm`、`success`、`failed`、`cancelled`、`compensating`。

## 2.4 OpenClaw 与 Hermes 的差异化适配

OpenClaw 和 Hermes 在架构理念和适用场景上存在差异，系统需要针对性适配。

### 2.4.1 适配差异对比

| 维度 | OpenClaw | Hermes |
|---|---|---|
| 核心理念 | 工具调用驱动 | 工作流编排驱动 |
| 适用场景 | 单次查询、即时问答、简单任务 | 多步骤任务、批量处理、复杂流程 |
| 交互模式 | 请求-响应，同步为主 | DAG 编排，支持同步和异步 |
| 工具粒度 | 细粒度，单一能力 | 粗粒度，组合能力 |
| 上下文管理 | 会话级，短期记忆 | 任务级，跨步骤状态传递 |
| 错误处理 | 单次失败即终止 | 支持重试、补偿和分支 |

### 2.4.2 OpenClaw 适配要点

- 工具描述需符合 OpenClaw 的 Tool Schema 规范
- 支持 OpenClaw 的 Function Calling 协议格式
- 工具返回结果需包含置信度和来源标注
- 写操作工具需支持 OpenClaw 的确认机制

### 2.4.3 Hermes 适配要点

- 提供 Workflow 节点定义接口，支持节点注册和状态查询
- 支持 Hermes 的 DAG 依赖关系定义
- 提供任务进度查询和状态回调机制
- 支持人工确认节点和异常补偿节点
- 长任务需支持断点续传和状态恢复

## 2.5 认证与授权机制设计

系统采用 Agent Token 和 API Key 双重认证方式。Agent 平台接入时分配唯一凭证，每次请求携带凭证进行身份校验。权限控制遵循最小权限原则，根据接入申请时声明的能力范围配置可调用工具列表和数据访问范围。Token 设置有效期，支持主动吊销和自动续期。所有工具调用前进行权限校验，越权请求直接拦截并记录审计日志。






