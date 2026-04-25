# dfkgo 后端实现计划

- 日期：2026-04-25
- 基于：`docs/superpowers/specs/2026-04-18-dfkgo-backend-design.md`
- 状态：执行中

---

## 总体策略

按依赖关系分 3 个阶段执行，共 4 个 Agent：

```
Phase 1 (串行)：Infrastructure Agent
    ↓
Phase 2 (并行)：Agent-A (Auth+User+File) ║ Agent-B (Task+History)
    ↓
Phase 3 (串行)：Lead 集成测试 + 打回修复
```

---

## Phase 1：基础设施（Infrastructure Agent）

### Scope

1. **依赖引入**：`go get` GORM + MySQL driver + SQLite driver(测试) + OSS SDK V2 + STS credentials + bcrypt
2. **`config/variables.go` 扩展**：新增所有 `app.env` 字段到 Config struct（DB、OSS、ModelServer、TaskWorker 相关）
3. **`errcode/` 包**：
   - `Error` 类型（code int, message string, 实现 error 接口）
   - `New(code, msg)` 构造函数
   - 所有错误码常量按段位定义（4011xx/4012xx/4013xx/4014xx/5000xx）
   - 包含 `IsErrCode(err) bool` 辅助函数
4. **`api/response/envelope.go`**：
   - `Envelope` struct：`Code int, Message string, Data any, Timestamp int64`
   - `OK(c, data)` / `OKWithMsg(c, msg, data)` / `Fail(c, code, msg)` / `FailWithErr(c, err)`
   - `FailWithErr` 自动识别 `errcode.Error` 提取 code/msg，非 errcode 错误走 500001
5. **`model/` 包**：3 个 GORM 模型
   - `User`：对应 users 表
   - `File`：对应 files 表，含 `FileUID` 字段
   - `Task`：对应 tasks 表，含 `TaskUID` 字段，`ResultJSON` 用 `datatypes.JSON`
6. **`repository/` 包**：3 个 repo
   - `UserRepo`：Create / FindByEmail / FindByID / UpdateProfile / UpdateAvatar
   - `FileRepo`：Create / FindByFileUID / FindByUserAndMD5 / UpdateUploadStatus / FindByID
   - `TaskRepo`：Create / FindByTaskUID / FindByTaskUIDAndUserID / UpdateStatus / UpdateResult / ListByUser(分页) / SoftDelete / BatchSoftDelete / CountByUser / FindPendingTasks / FindProcessingTasks / CountByUserAndModality / StatsForUser
7. **`service/oss/` 包**：
   - OSS Client 单例（`sync.Once`）
   - `IssueSTSCredentials(bucket, objectKeyPrefix, durationSec)` → STS 临时凭证
   - `HeadObject(bucket, objectKey)` → 校验文件存在
   - `BuildOssURL(bucket, objectKey)` → 拼完整 URL
8. **`auth/payload.go` 改造**：Payload 增加 `UserID int64` 字段，MakeToken/VerifyToken 需支持 UserID
9. **`api/` 目录重构**：
   - 创建 `api/handler/` 子目录
   - 创建 `api/middleware/` 子目录：迁移 jwt.go，新增 logger.go、recovery.go
   - `api/server.go` 重构：Server struct 持有 DB/config 等依赖，路由全部加 `/api` 前缀，设置路由组占位
   - 删除旧的 `api/auth.go`、`api/middleware.go`、`api/debug.go`，功能迁移到新目录
10. **DB 初始化**：在 server.go 或独立包中 GORM 连接 + AutoMigrate

### 成功标志

- [x] `go build ./...` 编译通过
- [x] `errcode.New()` 可构造错误，`FailWithErr` 正确提取 code
- [x] `response.OK/Fail` 输出符合 envelope 格式
- [x] 3 个 model AutoMigrate 到 SQLite 不报错
- [x] repository CRUD 单元测试通过（SQLite in-memory）
- [x] OSS client 可初始化（mock 模式下不连真实 OSS）
- [x] JWT payload 携带 UserID，现有 auth 测试通过
- [x] 路由前缀统一为 `/api`，health 端点 `GET /api/health` 可达
- [x] middleware/recovery 能兜底 panic 返回 5000xx envelope

### 关键约束

- model 只做 GORM 映射，禁止行为方法
- repository 禁止业务逻辑，只做 DB 读写
- 所有对外 ID 字段 (file_uid, task_uid) 在 model 中用 string，不暴露内部 BIGINT
- `response` 包不依赖 service/repository
- Config 用 Viper mapstructure tag，保持与 app.env 变量名一致
- STS 签发使用阿里云 credentials-go SDK 的 AssumeRole

---

## Phase 2A：Auth + User + File 域（Agent-A）

### 依赖

Phase 1 全部完成

### Scope

1. **`service/auth/` 包**：
   - `Register(email, password)` → bcrypt hash → repo 创建用户；重复邮箱抛 401009
   - `Login(email, password)` → repo 查用户 → bcrypt compare → JWT token + expiresAt；用户不存在 40104，密码错 401005
2. **`api/handler/auth.go`**：
   - `POST /api/auth/register`：参数校验（email 格式、密码 8-64 长度）→ 调 service → `response.OK`
   - `POST /api/auth/login`：参数校验 → 调 service → 返回 token+expiresAt
3. **`service/user/` 包**：
   - `GetProfile(userID)` → repo 查用户 → 返回 DTO
   - `UpdateProfile(userID, nickname, phone)` → repo 更新
   - `InitAvatarUpload(userID, mimeType, fileSize)` → 校验 mime/size → OSS STS → 返回凭证+objectKey
   - `AvatarUploadCallback(userID, objectKey)` → HeadObject 校验 → 更新 avatar_url
   - `FetchAvatar(userID)` → 返回 avatar_url
4. **`api/handler/user.go`**：
   - `GET /api/user/get-profile`
   - `PUT /api/user/update-profile`
   - `POST /api/user/avatar-upload/init`
   - `POST /api/user/avatar-upload/callback`
   - `GET /api/user/fetch-avatar`
5. **`service/file/` 包**：
   - `InitUpload(userID, fileName, fileSize, mimeType, md5)` → MD5 查重 → 命中返回 hit=true + fileId + ossUrl；未命中 → 创建 file 记录(pending) → STS 签发 → 返回 fileId + STS 凭证
   - `UploadCallback(userID, fileId)` → repo 查 file → HeadObject 校验 → 更新 upload_status=completed → 返回 ossUrl
   - mime 白名单校验、文件大小上限 500MB
   - modality 从 mimeType 推断（image/*, video/*, audio/*）
6. **`api/handler/upload.go`**：
   - `POST /api/upload/init`
   - `POST /api/upload/callback`
7. **路由注册**：在 server.go 中注册以上所有路由

### 成功标志

- [x] 注册 → 登录 → 拿 token 全流程 httptest 通过
- [x] 重复邮箱注册返回 401009
- [x] 错误密码登录返回 401005
- [x] get-profile 返回正确用户信息
- [x] update-profile 部分更新生效
- [x] avatar-upload/init 返回 STS 凭证结构（mock OSS）
- [x] upload/init 秒传命中返回 hit=true
- [x] upload/init 未命中返回 STS 凭证
- [x] upload/callback HeadObject 校验通过后 status=completed
- [x] 所有接口 envelope 格式正确
- [x] 未鉴权请求返回 401

### 关键约束

- handler 禁止含业务逻辑，只做参数解析+调 service+包装响应
- service 禁止持有 `*gin.Context`
- 密码用 bcrypt hash，不可明文存储
- 文件上传 STS objectKey 格式：`files/<userId>/<fileUid>.<ext>`
- 头像 STS objectKey 格式：`avatars/<userId>/<random>.<ext>`
- 头像 mimeType 白名单：image/jpg, image/jpeg, image/png；大小 ≤ 10MB
- 文件 mimeType 白名单见 api.md（video/mp4, video/quicktime, video/x-msvideo, audio/mpeg, audio/wav, image/jpeg, image/png）

---

## Phase 2B：Task + History 域（Agent-B）

### 依赖

Phase 1 全部完成

### Scope

1. **`service/task/queue.go`**：
   - `TaskQueue` 接口：`Push(ctx, taskID) error` / `Pop(ctx) (taskID, error)` / `Close() error`
   - `MemoryQueue` 实现：buffered channel，容量取 config `TASK_QUEUE_CAPACITY`（默认 1000）
2. **`service/task/modelclient.go`**：
   - `ModelClient` 接口：`Detect(ctx, modality, ossURL, taskID, userID) ([]byte, error)`
   - `HTTPModelClient` 实现：按 modality 路由到 `/api/detect/{image|video|audio}`，body `{oss_url, task_id, user_id}`，超时取 config
3. **`service/task/worker.go`**：
   - Worker pool：启动 N 个 goroutine（N = config `TASK_WORKER_POOL_SIZE`）
   - 每个 worker 循环：Pop → 查 DB 确认 status=pending → 置 processing+started_at → 调 ModelClient.Detect → 检查 status 是否 cancelled（丢弃结果）→ 写 result_json+status=completed+completed_at
   - ModelClient 错误 → status=failed + error_message
   - graceful shutdown：Close queue → 等待所有 worker 退出
4. **`service/task/service.go`**：
   - `CreateTask(userID, fileID, modality)` → 校验 file 存在+completed+属于 user → 创建 task(pending) → Push queue → 返回 taskId+status
   - `GetTaskStatus(userID, taskID)` → 查 task → 校验所属用户 → 返回状态 DTO
   - `GetTaskResult(userID, taskID)` → 查 task → 返回结果 DTO（含 result_json 透传）
   - `CancelTask(userID, taskID)` → 查 task → 校验状态可取消 → 置 cancelled
   - `RecoverOrphanTasks()` → processing→failed("service restarted") + pending→Push queue
5. **`api/handler/task.go`**：
   - `POST /api/tasks`
   - `GET /api/tasks/:taskId`
   - `GET /api/tasks/:taskId/result`
   - `POST /api/tasks/:taskId/cancel`
6. **`api/handler/history.go`**：
   - `GET /api/history` → 分页查询（page/limit），默认 page=1 limit=10，max limit=100
   - `DELETE /api/history/:taskId` → 单条软删
   - `POST /api/history/batch-delete` → 批量软删
   - `GET /api/history/stats` → 统计（total/byModality/byCategory）
7. **历史 service 逻辑**（可在 task service 中或独立 history service）：
   - `ListHistory(userID, page, limit)` → repo 分页查 + 总数 → 返回列表+分页信息
   - `DeleteHistory(userID, taskID)` → 软删（set deleted_at）
   - `BatchDeleteHistory(userID, taskIDs)` → 批量软删
   - `GetStats(userID)` → 聚合查询 total/byModality/byCategory
8. **Worker 生命周期集成**：server.go 中 Start 时启动 worker pool，shutdown 时优雅关闭
9. **路由注册**：在 server.go 中注册所有 task + history 路由

### 成功标志

- [x] 创建 task → status=pending，返回 taskId
- [x] fileId 不存在/不属于当前用户 → 返回错误码
- [x] worker 从 queue Pop → 调 mock ModelClient → task 状态变 completed + result_json 有值
- [x] cancel pending task → status=cancelled，worker Pop 后跳过
- [x] cancel processing task → status=cancelled，worker 完成后丢弃结果
- [x] cancel completed/failed/cancelled → 返回 401403
- [x] RecoverOrphanTasks：processing→failed，pending→重新入队
- [x] GET /api/tasks/:id 返回正确状态
- [x] GET /api/tasks/:id/result completed 时有 result，其他状态 result=null
- [x] GET /api/history 分页正确，软删记录不出现
- [x] DELETE + batch-delete 软删生效
- [x] GET /api/history/stats 聚合数据正确
- [x] ModelClient 返回非 200 → task failed + error_message
- [x] worker graceful shutdown 不丢数据

### 关键约束

- TaskQueue 是接口，MemoryQueue 为本期实现
- ModelClient 是接口，HTTPModelClient 为本期实现
- worker 每次 Pop 后必须再查 DB 确认 status 仍为 pending（防止并发取消）
- result_json 透传存储，不做 schema 校验
- 软取消语义：pending 直接 cancelled；processing 标记 cancelled 但不中断 ModelServer
- 任务 taskId 格式：`task_<32位hex>`
- 历史分页 deleted_at IS NULL 过滤
- stats 中 byCategory 需从 result_json 中提取 category 字段
- 所有 task 操作需校验 user_id 归属

---

## Phase 3：集成测试（Lead）

### Scope

1. 端到端 httptest 流程测试：注册 → 登录 → upload init → callback → 创建 task → 轮询 → 获取结果
2. 秒传流程测试
3. 历史分页 + 软删 + stats
4. 错误码覆盖
5. 并发安全验证

### 成功标志

- `go test ./...` 全部通过
- `go build ./...` 编译通过
- 所有 API envelope 格式一致
- 错误码正确返回

---

## 文件所有权矩阵（避免冲突）

| 文件/包 | Infrastructure | Agent-A | Agent-B | Lead |
|---|---|---|---|---|
| `errcode/` | **Owner** | R | R | R |
| `api/response/` | **Owner** | R | R | R |
| `config/` | **Owner** | R | R | R |
| `model/` | **Owner** | R | R | R |
| `repository/` | **Owner** | R | R | R |
| `service/oss/` | **Owner** | R | R | R |
| `auth/` | **Owner** | R | R | R |
| `api/server.go` | **Owner**(骨架) | **W**(注册路由) | **W**(注册路由) | W |
| `api/middleware/` | **Owner** | R | R | R |
| `api/handler/auth.go` | - | **Owner** | - | R |
| `api/handler/user.go` | - | **Owner** | - | R |
| `api/handler/upload.go` | - | **Owner** | - | R |
| `service/auth/` | - | **Owner** | - | R |
| `service/user/` | - | **Owner** | - | R |
| `service/file/` | - | **Owner** | - | R |
| `api/handler/task.go` | - | - | **Owner** | R |
| `api/handler/history.go` | - | - | **Owner** | R |
| `service/task/` | - | - | **Owner** | R |
| `main.go` | **Owner** | - | - | W |
| `entity/` | **Owner** | **W** | **W** | R |

> **Owner**=创建+完整维护  **W**=允许新增/修改  **R**=只读引用

---

## 风险与缓解

1. **server.go 并发修改**：Agent-A 和 Agent-B 都需注册路由。缓解：Infrastructure 先搭好路由组骨架（auth/user/upload/tasks/history），Agent-A/B 各自在对应组下添加 handler，减少冲突。若冲突由 Lead 合并。
2. **OSS/STS 无法真实测试**：所有 OSS 交互通过接口抽象，测试中 mock 实现。
3. **entity DTO 重复定义**：Agent-A 和 Agent-B 各自在 entity/ 创建不同文件，避免同文件冲突。
