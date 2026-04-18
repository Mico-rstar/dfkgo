# dfkgo 后端总体设计

- 日期：2026-04-18
- 状态：草案，待 review
- 适用范围：dfkgo 后端服务首期完整实现（用户、文件、任务、历史）
- 关联文档：
  - `VectorX.md`（项目背景与需求）
  - `docs/api.md`（dfkgo ↔ 前端 API 规范）
  - `docs/model_server_api.md`（ModelServer 现状）
  - `docs/aliyun_oss_go_sdk.md`（OSS SDK 用法）

---

## 1. 设计目标与范围

### 1.1 In Scope（本期实现）

- **用户域**：邮密注册（无邮箱验证激活）、邮密登录、Profile 查看与更新、头像 STS 上传
- **文件域**：STS 临时凭证直传 OSS、上传完成回调、用户维度 MD5 秒传
- **任务域**：统一 `/api/tasks` 接口创建检测任务，按模态（image / video / audio）分发至 ModelServer，状态查询、结果查询、软取消
- **历史域**：分页查询、单条 / 批量软删、统计接口
- **基础设施**：MySQL/GORM、OSS Go SDK V2 + STS、统一 envelope 响应、JWT 中间件、内存任务队列

### 1.2 Out of Scope（明确推迟到后续迭代）

- 邮箱验证激活、密码修改、邮箱修改、令牌刷新、登出失效（→ 不引入 Redis、不引入 SMTP）
- PDF 报告生成
- 多实例水平扩展、Redis / MQ 任务队列（架构上预留切换点）
- 任务真取消（依赖 ModelServer 提供取消接口）
- 历史记录多维筛选（仅分页，不按 modality / status / 时间筛）
- 跨模态（crossmodal）检测能力
- 进度百分比字段（ModelServer 不暴露进度，避免误导前端）

### 1.3 前置外部依赖

1. **ModelServer 接口改造**（dfkgo 不负责实现，作为外部约定）：
   - 提供按模态分类的 3 个端点：`/api/detect/image` `/api/detect/video` `/api/detect/audio`
   - 接收 `POST application/json {"oss_url": "...", "task_id": "...", "user_id": "..."}`
   - ModelServer 自行从 OSS 拉取源文件、检测、并**直接将结果图上传到 OSS `results/<userId>/<taskId>/`**
   - 返回结构与现有 `/api/detect` 同构，但 `mask_urls` / `masked_image_urls` 等**必须是 OSS URL**
2. **ModelServer OSS 权限**：独立 RAM 账号，对 dfkgo OSS Bucket：
   - `files/` 前缀：`oss:GetObject` / `oss:HeadObject` （读）
   - `results/` 前缀：`oss:PutObject` （写）
3. dfkgo 自身具备 OSS 长期 AK/SK + STS RAM Role（用于签发临时凭证给客户端直传）

### 1.4 设计原则

- **保持简单优先**：用户域、错误处理、配置都按最小可用集合落地，不为未确认的需求过度设计
- **解耦**：多模态文件全程经 OSS，dfkgo 不持久化文件字节流
- **接口稳定**：兼容 `docs/api.md` 已定义的路径与 envelope 响应
- **预留扩展点**：任务队列、ModelServer 客户端均通过接口抽象，便于未来切换实现

---

## 2. 系统架构

### 2.1 数据流总览

```
                    ┌─────────────────────────────────────┐
                    │         dfkgo (Gin, 单实例)         │
                    │                                     │
   客户端 ─HTTP────►│  api/  ──► service/  ──► repository │
                    │              │                ▲     │
                    │              ▼                │     │
                    │         taskqueue (内存)      │     │
                    │              │                │     │
                    │              ▼                │     │
                    │         worker pool ──┐       │     │
                    └───────────────────────┼───────┼─────┘
                                            │       │
                              HTTP JSON     │       │ GORM
                  ┌─────────────────────────┘       │
                  ▼                                 ▼
          ┌──────────────┐                  ┌──────────────┐
          │ ModelServer  │                  │   MySQL      │
          └─────┬────────┘                  └──────────────┘
                │ OSS GET (拉取文件)
                ▼
          ┌──────────────┐
          │  Aliyun OSS  │◄──── 客户端 PUT (STS 临时凭证直传)
          └──────────────┘
```

**连接说明**：
- dfkgo ↔ MySQL：GORM 读写（仅 dfkgo 访问）
- dfkgo → ModelServer：HTTP JSON（body 仅含 `oss_url + task_id`）
- ModelServer → OSS：ModelServer 主动拉取待检测文件
- 客户端 → OSS：STS 临时凭证直传
- **MySQL 与 ModelServer 无任何直接关联**

### 2.2 关键路径（一次完整检测）

1. 客户端 `POST /api/upload/init`（带 fileName/fileSize/mimeType/md5）
2. dfkgo 命中 MD5 → 直接返回已存在的 `fileId` + `oss_url`，跳过 3-4 步
3. dfkgo 未命中 → 调阿里云 STS 签发临时凭证，返回给客户端
4. 客户端用 STS 凭证直接 PUT 到 OSS
5. 客户端 `POST /api/upload/callback`，dfkgo `HeadObject` 校验 → files 表落库 `upload_status=completed`
6. 客户端 `POST /api/tasks {fileId, modality}` → dfkgo 创建 task 记录（status=pending）+ 推入内存队列
7. Worker goroutine `Pop` 任务 → 置 status=processing → 调 ModelServer `{oss_url, task_id}`
8. ModelServer 从 OSS 拉文件检测 → **将结果图（mask / masked_image）直接上传到 OSS `results/<userId>/<taskId>/` 路径** → 响应中返回 OSS URL
9. Worker 写回 `task.result_json` + `status=completed`（若途中被软取消则丢弃结果）
10. 客户端轮询 `GET /api/tasks/{id}` 与 `GET /api/tasks/{id}/result`（拿到的 mask URL 均为 OSS URL）

---

## 3. 模块结构

### 3.1 目录布局

```
dfkgo/
├── main.go
├── api/                      # HTTP 边界总包
│   ├── server.go             # Server struct + 路由注册 + worker 启停
│   ├── handler/              # 业务 handler：仅做参数解析 → 调 service → 包装响应
│   │   ├── auth.go           # /api/auth/*
│   │   ├── user.go           # /api/user/*
│   │   ├── upload.go         # /api/upload/*
│   │   ├── task.go           # /api/tasks/*
│   │   └── history.go        # /api/history/*
│   ├── middleware/           # 横切关注点
│   │   ├── jwt.go            # Bearer Token 鉴权
│   │   ├── logger.go         # 请求日志
│   │   └── recovery.go       # panic 兜底 + envelope 化为 5000xx 错误
│   └── response/             # 响应信封封装
│       ├── envelope.go       # OK / Fail / FailWithErr，errcode.Error 转 envelope
│       └── envelope_test.go
├── service/                  # 业务编排层
│   ├── auth/                 # 注册 / 登录
│   ├── user/                 # profile / 头像
│   ├── file/                 # 上传秒传 / 校验
│   ├── task/
│   │   ├── service.go        # 任务编排
│   │   ├── queue.go          # TaskQueue 接口 + MemoryQueue 实现
│   │   ├── worker.go         # worker pool
│   │   └── modelclient.go    # ModelServer HTTP 客户端 + 模态 adapter
│   └── oss/                  # OSS Client 单例 + STS 签发
├── repository/               # 数据访问层（GORM）
│   ├── user_repo.go
│   ├── file_repo.go
│   └── task_repo.go
├── model/                    # GORM 实体（无业务行为）
├── entity/                   # API 层 DTO
├── auth/                     # 现有 JWT 包，保留
├── config/                   # 现有配置，扩展 OSS / MySQL / ModelServer 字段
├── errcode/                  # 业务错误码常量 + Error 类型
└── docs/
    └── superpowers/specs/    # 设计文档
```

### 3.2 分层规则

| 层 | 职责 | 禁止 |
|---|---|---|
| `api/server.go` | Server 单例、路由注册、worker 启停 | 含业务逻辑 |
| `api/handler/` | 参数解析 → 调 service → 调 `response.OK/Fail` 包装响应 | 直接访问 GORM、含业务规则、手拼 envelope JSON |
| `api/middleware/` | 鉴权 / 日志 / Recovery 等横切能力 | 与具体业务耦合（如 hard-code 路由路径） |
| `api/response/` | envelope 封装 + `errcode.Error` 转 envelope | 依赖 service / repository |
| `service/` | 业务编排、调用 repository 与外部依赖；错误以 `errcode.Error` 抛出 | 直接持有 `*gin.Context`、返回 envelope 结构 |
| `repository/` | 仅 GORM 读写 | 含业务判断 |
| `model/` | GORM 表结构映射 | 含行为方法 |
| `entity/` | API DTO，与 model 不复用 | 暴露内部主键 |

**错误传递约定**：
- `service/` 抛 `errcode.Error`（6 位码 + msg）
- `api/handler/` 捕获后调 `response.FailWithErr(c, err)`，自动识别 `errcode.Error` 抽取 code/message
- `api/middleware/recovery.go` 兜底未捕获的 panic，转为 `5000xx` 内部错误 envelope

---

## 4. 数据模型

### 4.1 表结构

```sql
-- ============ users ============
CREATE TABLE users (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  email           VARCHAR(128)    NOT NULL,
  password_hash   VARCHAR(72)     NOT NULL,            -- bcrypt
  nickname        VARCHAR(64)     NOT NULL DEFAULT '',
  avatar_url      VARCHAR(512)    NOT NULL DEFAULT '',
  phone           VARCHAR(32)     NOT NULL DEFAULT '',
  created_at      DATETIME        NOT NULL,
  updated_at      DATETIME        NOT NULL,
  deleted_at      DATETIME        NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_email (email)
);

-- ============ files ============
CREATE TABLE files (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  file_uid        VARCHAR(40)     NOT NULL,            -- 对外 fileId, "file_<uuid>"
  user_id         BIGINT UNSIGNED NOT NULL,
  file_name       VARCHAR(255)    NOT NULL,
  mime_type       VARCHAR(64)     NOT NULL,
  file_size       BIGINT          NOT NULL,
  modality        VARCHAR(16)     NOT NULL,            -- image|video|audio
  md5             CHAR(32)        NOT NULL,
  oss_bucket      VARCHAR(64)     NOT NULL,
  oss_object_key  VARCHAR(512)    NOT NULL,
  oss_url         VARCHAR(1024)   NOT NULL,
  upload_status   VARCHAR(16)     NOT NULL,            -- pending|completed
  created_at      DATETIME        NOT NULL,
  updated_at      DATETIME        NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_file_uid (file_uid),
  UNIQUE KEY uk_user_md5 (user_id, md5),                -- 用户维度秒传
  KEY idx_user (user_id)
);

-- ============ tasks ============
CREATE TABLE tasks (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  task_uid        VARCHAR(40)     NOT NULL,            -- 对外 taskId, "task_<uuid>"
  user_id         BIGINT UNSIGNED NOT NULL,
  file_id         BIGINT UNSIGNED NOT NULL,
  modality        VARCHAR(16)     NOT NULL,
  status          VARCHAR(16)     NOT NULL,            -- pending|processing|completed|failed|cancelled
  result_json     JSON            NULL,
  error_message   VARCHAR(1024)   NOT NULL DEFAULT '',
  created_at      DATETIME        NOT NULL,
  started_at      DATETIME        NULL,
  completed_at    DATETIME        NULL,
  deleted_at      DATETIME        NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_task_uid (task_uid),
  KEY idx_user_history (user_id, deleted_at, created_at),
  KEY idx_file (file_id)
);
```

### 4.2 ID 规则

- DB 内部主键：`BIGINT UNSIGNED AUTO_INCREMENT`
- 对外 ID：UUID v4 去横线 + 业务前缀
  - 文件：`file_<32位hex>`
  - 任务：`task_<32位hex>`
- 所有跨表关联用内部数值主键，对外 API 仅暴露 `*_uid`

### 4.3 状态枚举

- `files.upload_status`：`pending` → `completed`（无失败状态，失败则 callback 不写库）
- `tasks.status`：`pending` → `processing` → (`completed` | `failed` | `cancelled`)
  - `cancelled` 可从 `pending` 或 `processing` 进入

---

## 5. API 规范

### 5.1 路径前缀与 envelope

- 所有路径以 `/api` 为前缀
- 所有响应统一信封：

```json
{
  "code": 0,
  "message": "Success",
  "data": { ... },
  "timestamp": 1620000000
}
```

- `code: 0` 表示成功；非 0 为业务错误码（6 位数字，沿用 `docs/api.md` 已定义段位）
- HTTP 状态码统一 `200`，仅在网关 / 框架级故障时使用 4xx/5xx

### 5.2 接口清单

| # | 路径 | 方法 | 鉴权 | 说明 |
|---|---|---|---|---|
| 1 | `/api/auth/register` | POST | ✗ | 邮密注册（无邮箱验证） |
| 2 | `/api/auth/login` | POST | ✗ | 邮密登录，签发 JWT（24h） |
| 3 | `/api/user/get-profile` | GET | ✓ | 返回 email / nickname / avatarUrl / phone |
| 4 | `/api/user/update-profile` | PUT | ✓ | 部分更新 nickname / phone |
| 5 | `/api/user/avatar-upload/init` | POST | ✓ | 头像 STS 凭证 |
| 6 | `/api/user/avatar-upload/callback` | POST | ✓ | 头像上传完成回调，更新 avatar_url |
| 7 | `/api/user/fetch-avatar` | GET | ✓ | 返回当前头像 URL |
| 8 | `/api/upload/init` | POST | ✓ | 文件 STS 凭证（含秒传命中） |
| 9 | `/api/upload/callback` | POST | ✓ | 文件上传完成回调 |
| 10 | `/api/tasks` | POST | ✓ | 创建检测任务 |
| 11 | `/api/tasks/{taskId}` | GET | ✓ | 任务状态 |
| 12 | `/api/tasks/{taskId}/result` | GET | ✓ | 任务结果（按 modality 异构 result） |
| 13 | `/api/tasks/{taskId}/cancel` | POST | ✓ | 软取消 |
| 14 | `/api/history` | GET | ✓ | 分页历史 (page / limit) |
| 15 | `/api/history/{taskId}` | DELETE | ✓ | 单条软删 |
| 16 | `/api/history/batch-delete` | POST | ✓ | 批量软删 |
| 17 | `/api/history/stats` | GET | ✓ | 统计：总数 / 各模态计数 / 各结论计数 |

### 5.3 关键接口请求/响应示例

#### 5.3.1 `POST /api/upload/init`

请求：

```json
{
  "fileName": "video.mp4",
  "fileSize": 104857600,
  "mimeType": "video/mp4",
  "md5": "9e107d9d372bb6826bd81d3542a419d6"
}
```

响应（未命中秒传，签发 STS）：

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "fileId": "file_8e5b...",
    "hit": false,
    "sts": {
      "accessKeyId": "STS.xxx",
      "accessKeySecret": "xxx",
      "securityToken": "xxx",
      "expiration": "2026-04-18 10:00:49",
      "bucket": "dfkgo-files",
      "region": "cn-hangzhou",
      "endpoint": "oss-cn-hangzhou.aliyuncs.com",
      "objectKey": "files/<userId>/file_8e5b....mp4"
    }
  },
  "timestamp": 1620000000
}
```

响应（命中秒传）：

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "fileId": "file_8e5b...",
    "hit": true,
    "ossUrl": "https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/files/..."
  },
  "timestamp": 1620000000
}
```

#### 5.3.2 `POST /api/tasks`

请求：

```json
{ "fileId": "file_8e5b...", "modality": "video" }
```

响应：

```json
{
  "code": 0,
  "message": "Success",
  "data": { "taskId": "task_345...", "status": "pending" },
  "timestamp": 1620000000
}
```

#### 5.3.3 `GET /api/tasks/{taskId}/result`

响应（image 模态示例，mask URL 均为 dfkgo 转存后的 OSS URL）：

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "taskId": "task_345...",
    "modality": "image",
    "status": "completed",
    "result": {
      "category": "tampered",
      "description": "...",
      "maskUrls": ["https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/results/123/task_345.../mask_0.jpg"],
      "maskedImageUrls": ["https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/results/123/task_345.../masked_0.jpg"]
    },
    "completedAt": 1620003600
  },
  "timestamp": 1620003600
}
```

video / audio 的 `result` 字段结构由 ModelServer 实际返回决定，dfkgo 透传 `result_json` 内容。**所有图片 / 媒体 URL 由 ModelServer 直接生成为 OSS URL**（ModelServer 在返回前已将结果图上传到约定的 OSS 路径），dfkgo 不做 URL 转存与改写。

### 5.4 错误码

- `errcode` 包统一定义常量，沿用 `docs/api.md` 已存在的 `401005 / 40104 / 401009 / 401018` 等
- 新增段位规划：
  - `4011xx`：认证 / 授权（如 `401005` 密码错误、`401009` 用户已存在）
  - `4012xx`：用户域（profile / 头像）
  - `4013xx`：文件域（如 `401018` 文件已存在 / 秒传命中）
  - `4014xx`：任务域（任务不存在、状态不允许取消等）
  - `5000xx`：内部错误（DB / OSS / ModelServer 异常）
- 错误码与 message 一一对应，handler 中通过 `errcode.New(code)` 构造
- 实施期由 `errcode` 包按上述段位逐个定义常量，避免散落在 handler 内

---

## 6. 异步任务流水线

### 6.1 TaskQueue 抽象

```go
type TaskQueue interface {
    Push(ctx context.Context, taskID string) error
    Pop(ctx context.Context) (taskID string, err error)
    Close() error
}
```

本期实现：`MemoryQueue`（带 buffer 的 channel，容量取自 `TASK_QUEUE_CAPACITY`，默认 1000）。
未来扩展：`RedisQueue` 实现，支持多实例消费。

### 6.2 Worker pool

- 启动时拉起 `TASK_WORKER_POOL_SIZE` 个 goroutine（默认 4）
- 每个 worker 循环：`Pop → 检查 status 是否仍为 pending → 置 processing → 调 ModelServer → 写库`
- ModelServer 调用超时：`MODEL_SERVER_TIMEOUT_SECONDS`，默认 300s

### 6.3 启动时孤儿任务恢复

dfkgo 进程启动时执行：
1. 扫描 `tasks WHERE status = 'processing'` → 全部置为 `failed`，`error_message = "service restarted"`（保守策略：重启不可恢复中间态）
2. 扫描 `tasks WHERE status = 'pending'` → 重新 `Push` 入队

### 6.4 ModelServer 客户端

```go
type ModelClient interface {
    Detect(ctx context.Context, modality Modality, ossURL string, taskID string) (resultJSON []byte, err error)
}
```

按 modality 路由：
- `image` → `POST {base}/api/detect/image`
- `video` → `POST {base}/api/detect/video`
- `audio` → `POST {base}/api/detect/audio`

请求体统一：`{"oss_url": "...", "task_id": "..."}`
响应体：透传字节，落入 `tasks.result_json`

### 6.5 软取消语义

| 当前状态 | cancel 行为 |
|---|---|
| `pending` | 直接置 `cancelled`；worker `Pop` 后检查状态发现非 pending，跳过 |
| `processing` | 置 `cancelled`；ModelServer 仍跑完，worker 收到结果后**丢弃**（不写 result_json） |
| `completed` / `failed` / `cancelled` | 拒绝，返回业务错误码 |

未来 ModelServer 提供取消接口后，可在此扩展真取消逻辑。

### 6.6 结果图片持久化

**由 ModelServer 直接上传结果到 OSS**，dfkgo 不接触结果字节流。

**约定**：
- ModelServer 拥有 dfkgo OSS Bucket 的**写权限**（`oss:PutObject`），写入路径限定在 `results/` 前缀
- ModelServer 检测完成后：
  1. 生成 mask / masked_image 本地临时文件
  2. PutObject 到 `dfkgo-files/results/<userId>/<taskId>/<原文件名>`（`task_id` 已随请求传入，`userId` 可从 task_id 反查或者由 dfkgo 额外传入，见 model_server_api.md）
  3. 在响应 `mask_urls` / `masked_image_urls` 中返回该 OSS URL（完整 https 地址）
- dfkgo worker 拿到响应后**原样透传** `result_json` 入库，不做转存与改写

```
worker.run(task):
    raw = modelClient.Detect(task.modality, task.file.oss_url, task.id, task.user_id)
    task.result_json = raw   // URL 已是 OSS URL
    task.status = completed; save()
```

**优点**：
- dfkgo 真零字节流，严格遵守文件解耦 OSS 原则
- 历史记录与结果图生命周期一致（不依赖 ModelServer 本地文件保留期）
- 不暴露 ModelServer 域名

**代价与须知**：
- 对 ModelServer 改造要求提高（需接入 OSS SDK + 获得写权限 RAM 账号）——这是**前置外部依赖**，列入 1.3 节
- ModelServer OSS 写入失败时：应返回 `502` + `detail`，dfkgo 接收后 task 置 `failed`，`error_message` 记录原因
- ModelServer 部分图片上传失败策略：由 ModelServer 决定（建议该 URL 返回空字符串，dfkgo 不判断，透传即可）

**未来升级路径**：如需加强安全（避免 OSS URL 泄露即可访问），可要求 ModelServer 存 objectKey 而非完整 URL，dfkgo 在响应 `/api/tasks/{id}/result` 时动态 Presign GET URL。

---

## 7. 配置

### 7.1 `app.env` 字段

```
SERVER_PORT=:8888
JWT_PRIVATE_KEY=xxx
JWT_DURATION_HOURS=24

DB_DRIVER=mysql
DB_SOURCE=user:pwd@tcp(host:3306)/dfkgo?charset=utf8mb4&parseTime=true&loc=Local

OSS_REGION=cn-hangzhou
OSS_ENDPOINT=                     # 可选，留空走默认
OSS_BUCKET_FILES=dfkgo-files
OSS_BUCKET_AVATARS=dfkgo-avatars
OSS_ACCESS_KEY_ID=xxx
OSS_ACCESS_KEY_SECRET=xxx

OSS_STS_ROLE_ARN=acs:ram::<uid>:role/dfkgo-uploader
OSS_STS_DURATION_SECONDS=900

MODEL_SERVER_BASE_URL=https://...
MODEL_SERVER_TIMEOUT_SECONDS=300

TASK_WORKER_POOL_SIZE=4
TASK_QUEUE_CAPACITY=1000
```

### 7.2 Bucket / 路径规划

- `dfkgo-files/files/<userId>/<fileUid>.<ext>` —— 检测源文件（客户端 STS 直传）
- `dfkgo-files/results/<userId>/<taskId>/<原文件名>` —— **ModelServer 直传**的检测结果图（mask / masked_image）
- `dfkgo-avatars/avatars/<userId>/<random>.<ext>` —— 头像

**权限划分**：
- dfkgo：全路径读写（含 STS 签发、HeadObject）
- ModelServer：独立 RAM 账号，`files/` 前缀只读，`results/` 前缀可写
- 客户端：STS 临时凭证写 `files/<userId>/`、头像写 `avatars/<userId>/`

OSS 控制台为 `tmp/` 前缀配置生命周期规则（如 7 天自动清理失败上传的临时对象）。`results/` 前缀不设生命周期，后续可加定时任务清理 `deleted_at` 超过 N 天的 `results/<userId>/<taskId>/` 路径。

---

## 8. 错误处理与日志

- 业务错误统一通过 `errcode.New(code, msg)` 抛出，全局中间件捕获并 envelope 输出
- ModelServer / OSS 调用失败：捕获 `*oss.ServiceError` 与 HTTP 4xx/5xx，落到 `task.error_message`，task 置 `failed`
- 日志：使用 gin 默认 logger 中间件 + 关键节点结构化打日志
  - INFO：task 创建 / 完成 / 失败、ModelServer 调用 RT
  - ERROR：OSS / ModelServer 调用异常、DB 异常
- 不引入额外日志框架（本期保持简单），后续如需结构化日志再引入 zap / zerolog

---

## 9. 测试策略

- **单元测试**：service 层 mock repository 与外部依赖（OSS Client、ModelClient）
- **repository 测试**：用 sqlite in-memory 替代 MySQL 跑 GORM 验证查询正确性
- **HTTP 集成测试**：用 `httptest` 启动 Server，外部依赖 mock 后跑端到端
- **OSS / ModelServer mock**：`httptest.Server` 模拟响应
- 现有 `auth/` 包测试保留并扩展
- 不做真实环境 e2e（本期）

---

## 10. 实施分阶段（供后续 plan 参考）

1. **Phase 1：基础设施**
   - 引入 GORM + MySQL 连接池、`errcode` 包、`api/response.go` envelope 封装
   - 实现 `service/oss`：OSS Client 单例 + STS 签发
   - `service/task/modelclient.go` skeleton（接口 + 空实现）
2. **Phase 2：用户域**
   - register / login / profile / 头像 STS + callback
3. **Phase 3：文件域**
   - upload init（含 MD5 秒传）/ callback / `HeadObject` 校验
4. **Phase 4：任务域**
   - `MemoryQueue` + worker pool + 任务 CRUD + ModelServer 调用（透传含 OSS URL 的结果）+ 软取消 + 启动时孤儿任务恢复
5. **Phase 5：历史域**
   - 分页查询 / 单条 + 批量软删 / stats 聚合查询

每个 Phase 完成后跑测试 + 手动联调。详细任务拆解由后续 plan 文档负责。

---

## 11. 风险与开放问题

| 风险 / 问题 | 描述 | 缓解 / 决策 |
|---|---|---|
| ModelServer 接口未改造 | 当前 `/api/detect` 仍是 multipart，不支持 OSS URL | 列入前置依赖，与 ModelServer 团队同步排期；dfkgo 客户端层用接口抽象，未来切实现即可 |
| 单实例任务队列 | 进程崩溃丢失内存中的 pending 任务 | 启动时扫库重新入队；多实例需求出现时切 Redis 实现 |
| STS 凭证泄漏 | 客户端拿到 STS 后被截获 | TTL 短（默认 900s）+ RAM Role 严格限定到 bucket 与 path 前缀 |
| 秒传 MD5 碰撞 | 理论存在但概率极低 | 接受；如需防御可扩展为 MD5 + 文件大小双索引 |
| ModelServer 超长任务 | 视频检测耗时可能超过 5min 默认超时 | `MODEL_SERVER_TIMEOUT_SECONDS` 可配；后续可按 modality 设差异化超时 |
| ModelServer OSS 写入失败 | ModelServer 拿到结果后上传 OSS 失败 | ModelServer 返回 `502` + `detail`；dfkgo task 置 `failed`，`error_message` 记录原因 |
| ModelServer 权限越界 | ModelServer RAM 账号写到 `files/` 以外路径 | RAM Policy 严格限定 `oss:PutObject` 仅作用于 `results/` 前缀 |
| OSS results/ 前缀无生命周期 | 历史软删后结果图未被删除 | 本期接受；后续可加定时任务清理 `deleted_at` 超过 N 天的 `results/<userId>/<taskId>/` 路径 |

---

## 12. 决策记录（来自 brainstorming）

| 编号 | 决策 |
|---|---|
| Q1 | 检测能力支持 image / video / audio 三模态（按 ModelServer 实际能力） |
| Q2 | 统一任务接口 + 按模态区分异构结果体（discriminated union） |
| Q3 | 内存任务队列起步；砍掉百分比进度，仅状态枚举 |
| Q4 | STS 直传 OSS + 客户端主动 callback；ModelServer 从 OSS 自行拉数据，dfkgo 不中转字节流 |
| Q5 | 4 张表精简为 3 张（users / files / tasks），无 Redis；不实现 JWT 失效；对外 ID 用 uuid 字符串 |
| Q6 | 用户域最小化：仅 register / login / profile / avatar；砍掉邮箱验证、改密码、改邮箱、refresh、logout |
| Q7 | API 路径前缀 `/api`（按 docs/api.md） |
| Q8 | 强制 envelope 响应：`code:0` 成功 + 6 位业务错误码 |
| Q9 | OSS 上传保留 STS 临时凭证方案（前端已实现，迁移成本大） |
| Q10 | 移除 `/api/upload/init` 响应中的 `chunkSize` 字段 |
| Q11 | 实现用户维度 MD5 秒传（`(user_id, md5)` 唯一索引） |
| Q12 | 历史记录仅分页，不加多维筛选 |
| Q13 | 实现 `/api/history/stats` 统计接口 |
| Q14 | PDF 报告生成不在本期范围 |
| Q15 | 软取消：pending 真取消；processing 标记 cancelled 但 ModelServer 仍跑完，结果丢弃 |
| Q17 | **结果图片由 ModelServer 直接上传 OSS**，dfkgo 不转存不改写；ModelServer 需额外获得 `results/` 写权限 |
| Q16 | 单实例部署为主，预留 Redis 队列切换点（接口抽象 `TaskQueue`） |
