# dfkgo ↔ 客户端 RESTful API 规范

> 本文档描述 dfkgo 后端对前端 / 客户端暴露的接口契约。
>
> 关联文档：
> - 设计文档：`docs/superpowers/specs/2026-04-18-dfkgo-backend-design.md`
> - dfkgo ↔ ModelServer：`docs/model_server_api.md`
> - OSS SDK 用法：`docs/aliyun_oss_go_sdk.md`

---

## 基础信息

- **Base URL**：`https://api.example.com`（实际地址按部署环境）
- **路径前缀**：所有业务接口以 `/api` 开头
- **数据格式**：`Content-Type: application/json`
- **认证方式**：Bearer Token（JWT），Header 形式：`Authorization: Bearer <token>`
- **JWT 有效期**：24 小时；本期不实现刷新 / 失效机制，到期重新登录

---

## 统一响应格式（Envelope）

**所有 dfkgo 接口** 均返回统一信封结构，HTTP 状态码统一 `200`（仅在网关级故障时返回 4xx/5xx）。

### 成功

```json
{
  "code": 0,
  "message": "Success",
  "data": { ... },
  "timestamp": 1620000000
}
```

### 失败

```json
{
  "code": 401005,
  "message": "用户密码错误，请重新输入",
  "data": null,
  "timestamp": 1620000000
}
```

### 业务错误码段位

| 段位 | 含义 |
|---|---|
| `0` | 成功 |
| `4011xx` | 认证 / 授权（如 `401005` 密码错误、`401009` 用户已存在、`40104` 用户不存在） |
| `4012xx` | 用户域（profile / 头像） |
| `4013xx` | 文件域（如 `401018` 文件已存在 / 秒传命中） |
| `4014xx` | 任务域（任务不存在、状态不允许操作等） |
| `5000xx` | 内部错误（DB / OSS / ModelServer 异常） |

---

## 1. 认证模块

### 1.1 用户注册

- **`POST /api/auth/register`**
- **认证**：✗
- **说明**：邮箱 + 密码注册。无邮箱验证激活步骤，注册即可登录。

**请求体**

```json
{
  "email": "user@example.com",
  "password": "Password123"
}
```

| 字段 | 必填 | 校验 |
|---|---|---|
| email | 是 | 合法邮箱格式，全局唯一 |
| password | 是 | 长度 8~64 |

**成功响应**

```json
{ "code": 0, "message": "注册成功", "data": null, "timestamp": 1620000000 }
```

**典型错误**

| code | message |
|---|---|
| 401009 | 用户已存在 |
| 400001 | 邮箱格式不合法 |
| 400002 | 密码不符合规则 |

---

### 1.2 用户登录

- **`POST /api/auth/login`**
- **认证**：✗

**请求体**

```json
{
  "email": "user@example.com",
  "password": "Password123"
}
```

**成功响应**

```json
{
  "code": 0,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresAt": 1620086400
  },
  "timestamp": 1620000000
}
```

**典型错误**

| code | message |
|---|---|
| 40104 | 用户不存在 |
| 401005 | 用户密码错误，请重新输入 |

> **本期不实现** 邮箱验证激活、令牌刷新、登出失效、密码修改、邮箱修改。规划在后续迭代。

---

## 2. 用户模块

所有接口均需 JWT 鉴权。

### 2.1 获取个人信息

- **`GET /api/user/get-profile`**

**成功响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "email": "user@example.com",
    "nickname": "new_user",
    "avatarUrl": "https://dfkgo-avatars.oss-cn-hangzhou.aliyuncs.com/avatars/123/xxx.jpg",
    "phone": ""
  },
  "timestamp": 1620000000
}
```

---

### 2.2 更新个人信息

- **`PUT /api/user/update-profile`**
- **说明**：部分更新；不需要更新的字段可省略。

**请求体**

```json
{
  "nickname": "新昵称",
  "phone": "13800138000"
}
```

**成功响应**

```json
{ "code": 0, "message": "用户信息更新成功", "data": null, "timestamp": 1620000000 }
```

---

### 2.3 头像上传初始化（STS）

- **`POST /api/user/avatar-upload/init`**
- **说明**：签发阿里云 STS 临时凭证，客户端用此凭证将头像直传到 OSS。

**请求体**

```json
{
  "mimeType": "image/jpeg",
  "fileSize": 102400
}
```

| 字段 | 必填 | 校验 |
|---|---|---|
| mimeType | 是 | 必须为 `image/jpg` / `image/jpeg` / `image/png` |
| fileSize | 是 | 字节，≤ 10MB |

**成功响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "accessKeyId": "STS.NXarhQXK7KQAABHkNK2oue11c",
    "accessKeySecret": "FqNNhsuiwusAk38fiJa7zfpCRzjoDgP2YQbvH6JzzK5Y",
    "securityToken": "CAISnAN1q6Ft5B...",
    "expiration": "2026-04-18 10:00:49",
    "bucket": "dfkgo-avatars",
    "region": "cn-hangzhou",
    "endpoint": "oss-cn-hangzhou.aliyuncs.com",
    "objectKey": "avatars/123/avatar_0f760b557422adac.jpg",
    "maxFileSize": 10485760,
    "allowedFileTypes": ["jpg", "jpeg", "png"]
  },
  "timestamp": 1620000000
}
```

---

### 2.4 头像上传完成回调

- **`POST /api/user/avatar-upload/callback`**
- **说明**：客户端将头像 PUT 到 OSS 成功后调此接口，dfkgo `HeadObject` 校验后更新 `users.avatar_url`。

**请求体**

```json
{
  "objectKey": "avatars/123/avatar_0f760b557422adac.jpg"
}
```

**成功响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "avatarUrl": "https://dfkgo-avatars.oss-cn-hangzhou.aliyuncs.com/avatars/123/avatar_0f760b557422adac.jpg"
  },
  "timestamp": 1620000000
}
```

**典型错误**

| code | message |
|---|---|
| 401202 | 文件未在 OSS 上找到 |

---

### 2.5 获取头像 URL

- **`GET /api/user/fetch-avatar`**

**成功响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "avatarUrl": "https://dfkgo-avatars.oss-cn-hangzhou.aliyuncs.com/avatars/123/avatar.jpg"
  },
  "timestamp": 1620000000
}
```

---

## 3. 文件上传模块

所有接口均需 JWT 鉴权。

### 3.1 上传初始化（STS + 秒传）

- **`POST /api/upload/init`**
- **说明**：签发 STS 凭证供客户端直传 OSS。如命中用户维度 MD5 秒传，则直接返回已存在的 fileId，跳过实际上传。

**请求体**

```json
{
  "fileName": "video.mp4",
  "fileSize": 104857600,
  "mimeType": "video/mp4",
  "md5": "9e107d9d372bb6826bd81d3542a419d6"
}
```

| 字段 | 必填 | 校验 |
|---|---|---|
| fileName | 是 | 含扩展名 |
| fileSize | 是 | ≤ 500 MB |
| mimeType | 是 | 白名单：`video/mp4`、`video/quicktime`（mov）、`video/x-msvideo`（avi）、`audio/mpeg`（mp3）、`audio/wav`、`image/jpeg`、`image/png` |
| md5 | 是 | 32 位小写 hex，文件级 MD5 |

**成功响应（未命中秒传，签发 STS）**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "fileId": "file_8e5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d",
    "hit": false,
    "sts": {
      "accessKeyId": "STS.xxx",
      "accessKeySecret": "xxx",
      "securityToken": "xxx",
      "expiration": "2026-04-18 10:00:49",
      "bucket": "dfkgo-files",
      "region": "cn-hangzhou",
      "endpoint": "oss-cn-hangzhou.aliyuncs.com",
      "objectKey": "files/123/file_8e5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d.mp4"
    }
  },
  "timestamp": 1620000000
}
```

**成功响应（命中秒传）**

```json
{
  "code": 0,
  "message": "文件已存在",
  "data": {
    "fileId": "file_8e5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d",
    "hit": true,
    "ossUrl": "https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/files/123/..."
  },
  "timestamp": 1620000000
}
```

> **秒传命中时**，客户端可直接进入 `POST /api/tasks` 创建检测任务，**跳过 PUT OSS 与 callback 步骤**。

**典型错误**

| code | message |
|---|---|
| 401301 | 文件类型不支持 |
| 401302 | 文件大小超出限制 |

---

### 3.2 上传完成回调

- **`POST /api/upload/callback`**
- **说明**：客户端用 STS 凭证 PUT 到 OSS 成功后调此接口，dfkgo `HeadObject` 校验后将 files 记录置 `upload_status=completed`。

**请求体**

```json
{ "fileId": "file_8e5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d" }
```

**成功响应**

```json
{
  "code": 0,
  "message": "文件上传完成",
  "data": {
    "fileId": "file_8e5b...",
    "ossUrl": "https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/files/123/..."
  },
  "timestamp": 1620000000
}
```

**典型错误**

| code | message |
|---|---|
| 401303 | 文件未在 OSS 上找到 |
| 401304 | fileId 不存在 |

---

## 4. 检测任务模块

所有接口均需 JWT 鉴权。

### 4.1 创建检测任务

- **`POST /api/tasks`**
- **说明**：创建检测任务，dfkgo 内部按 `modality` 分发到 ModelServer。任务异步执行，立即返回 `pending` 状态。

**请求体**

```json
{
  "fileId": "file_8e5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d",
  "modality": "video"
}
```

| 字段 | 必填 | 取值 |
|---|---|---|
| fileId | 是 | `/api/upload/init` 或 `/api/upload/callback` 返回的 fileId |
| modality | 是 | `image` / `video` / `audio` |

**成功响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "taskId": "task_345678abcdef9012345678abcdef9012",
    "status": "pending"
  },
  "timestamp": 1620000000
}
```

**典型错误**

| code | message |
|---|---|
| 401304 | fileId 不存在 |
| 401401 | modality 与文件类型不匹配 |

---

### 4.2 获取任务状态

- **`GET /api/tasks/{taskId}`**
- **说明**：轮询接口，返回当前任务状态。**不返回进度百分比**（ModelServer 不暴露进度）。

**成功响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "taskId": "task_345678...",
    "status": "processing",
    "modality": "video",
    "createdAt": 1620000000,
    "startedAt": 1620000005,
    "completedAt": null
  },
  "timestamp": 1620000010
}
```

**`status` 枚举**：`pending` / `processing` / `completed` / `failed` / `cancelled`

---

### 4.3 获取任务结果

- **`GET /api/tasks/{taskId}/result`**
- **说明**：仅当 `status=completed` 时 `result` 字段有值；其他状态时 `result` 为 `null`。
- **结果体按 modality 异构**，`result` 内字段透传自 ModelServer 响应。

**成功响应（image 模态示例）**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "taskId": "task_345678...",
    "modality": "image",
    "status": "completed",
    "result": {
      "category": "tampered",
      "description": "The tampering involves the alteration of a single object...",
      "maskUrls": ["https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/results/123/task_345.../mask_0.jpg"],
      "maskedImageUrls": ["https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/results/123/task_345.../masked_0.jpg"]
    },
    "completedAt": 1620003600
  },
  "timestamp": 1620003600
}
```

**video / audio 模态**：`result` 字段结构由 ModelServer 实际返回决定，dfkgo 透传 `result_json`。客户端按 `modality` 字段判别后做相应渲染。

**失败任务响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "taskId": "task_345678...",
    "modality": "video",
    "status": "failed",
    "result": null,
    "errorMessage": "modelserver 502: failed to fetch oss_url",
    "completedAt": 1620003600
  },
  "timestamp": 1620003600
}
```

**典型错误**

| code | message |
|---|---|
| 401402 | 任务不存在 |

---

### 4.4 取消任务（软取消）

- **`POST /api/tasks/{taskId}/cancel`**
- **语义说明**：
  - `pending` 状态：直接取消，worker 出队时跳过
  - `processing` 状态：标记为 `cancelled`，**ModelServer 仍会跑完**（依赖 ModelServer 后续提供取消接口才能真取消），但 worker 收到结果后丢弃
  - `completed` / `failed` / `cancelled` 状态：返回错误

**成功响应**

```json
{ "code": 0, "message": "任务已取消", "data": null, "timestamp": 1620000000 }
```

**典型错误**

| code | message |
|---|---|
| 401402 | 任务不存在 |
| 401403 | 任务状态不允许取消 |

---

## 5. 历史记录模块

所有接口均需 JWT 鉴权。

### 5.1 历史记录列表（分页）

- **`GET /api/history?page=1&limit=10`**
- **说明**：仅支持分页，**不支持按时间 / modality / status 等多维筛选**（本期 Out of Scope）。

**查询参数**

| 字段 | 必填 | 默认 | 说明 |
|---|---|---|---|
| page | 否 | 1 | 页码，从 1 开始 |
| limit | 否 | 10 | 每页条数，最大 100 |

**成功响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "items": [
      {
        "taskId": "task_345678...",
        "fileName": "video.mp4",
        "modality": "video",
        "fileSize": 104857600,
        "status": "completed",
        "createdAt": 1620000000,
        "completedAt": 1620003600
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 45,
      "pages": 5
    }
  },
  "timestamp": 1620000000
}
```

> 已软删 (`deleted_at IS NOT NULL`) 的任务不出现在列表中。

---

### 5.2 删除单条历史

- **`DELETE /api/history/{taskId}`**
- **说明**：软删（更新 `deleted_at`）。

**成功响应**

```json
{ "code": 0, "message": "记录删除成功", "data": null, "timestamp": 1620000000 }
```

---

### 5.3 批量删除

- **`POST /api/history/batch-delete`**

**请求体**

```json
{ "taskIds": ["task_123", "task_456", "task_789"] }
```

**成功响应**

```json
{
  "code": 0,
  "message": "批量删除成功",
  "data": { "deletedCount": 3 },
  "timestamp": 1620000000
}
```

---

### 5.4 数据统计

- **`GET /api/history/stats`**
- **说明**：当前用户已完成（未软删）任务的聚合统计。

**成功响应**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "total": 45,
    "byModality": {
      "image": 20,
      "video": 18,
      "audio": 7
    },
    "byCategory": {
      "real": 30,
      "tampered": 12,
      "fake": 3
    }
  },
  "timestamp": 1620000000
}
```

| 字段 | 说明 |
|---|---|
| total | 用户已完成检测任务总数 |
| byModality | 各模态计数 |
| byCategory | 各检测结论计数（基于 `tasks.result_json.category`） |

---

## 6. 完整调用流程示例

### 6.1 普通检测流程（视频，未命中秒传）

```
1. POST /api/auth/login              → 拿到 token
2. POST /api/upload/init              → 拿到 fileId + STS 凭证
3. PUT  <ossEndpoint>/<objectKey>     → 客户端 OSS SDK 直传（dfkgo 不参与）
4. POST /api/upload/callback          → dfkgo 校验 OSS 文件存在
5. POST /api/tasks                    → 拿到 taskId, status=pending
6. GET  /api/tasks/{taskId}  (loop)   → 轮询直至 status=completed/failed
7. GET  /api/tasks/{taskId}/result    → 获取最终结果
```

### 6.2 秒传命中流程

```
1. POST /api/auth/login               → 拿到 token
2. POST /api/upload/init              → hit=true, 直接拿到 fileId + ossUrl
3. POST /api/tasks                    → 拿到 taskId（跳过 OSS PUT 与 callback）
4. GET  /api/tasks/{taskId}  (loop)
5. GET  /api/tasks/{taskId}/result
```

---

## 7. 本期 Out of Scope

以下接口暂不实现，规划在后续迭代：

- `POST /api/auth/verify-email`（邮箱验证激活）
- `POST /api/auth/refresh-token`（令牌刷新）
- `POST /api/auth/logout`（登出失效）
- `PUT /api/user/password`（修改密码）
- `POST /api/user/email/request-change` + `confirm-change`（修改邮箱）
- `POST /api/reports/{taskId}`（PDF 报告生成）
- 历史记录多维筛选（modality / status / 时间区间）
