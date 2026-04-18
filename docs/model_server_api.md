# dfkgo ↔ ModelServer API 规范

> 本文档描述 dfkgo 后端调用 ModelServer 的接口契约。
> ModelServer 是常驻 GPU 服务：
> 1. **主动从 OSS 拉取待检测文件**（读 `files/` 前缀）
> 2. 检测完成后**直接将结果图（mask / masked_image）上传到 OSS**（写 `results/` 前缀）
> 3. 响应中返回已上传的 OSS URL
>
> dfkgo 全程不接触任何文件字节流，仅传递元数据。
>
> 关联文档：
> - 设计文档：`docs/superpowers/specs/2026-04-18-dfkgo-backend-design.md`
> - dfkgo ↔ 客户端：`docs/api.md`

---

## 基础信息

- **Base URL**：由 dfkgo 配置 `MODEL_SERVER_BASE_URL` 注入，如 `https://model.example.com`
- **协议**：HTTP/HTTPS
- **请求格式**：`Content-Type: application/json`
- **响应格式**：`Content-Type: application/json`
- **超时**：dfkgo 端默认 300s，由 `MODEL_SERVER_TIMEOUT_SECONDS` 控制
- **鉴权**：本期暂不要求（内网部署）；如需扩展可后续加 API Key Header

---

## 1. 健康检查

### `GET /health`

用于 dfkgo 启动 / 巡检时探活。

**请求示例**

```bash
curl https://model.example.com/health
```

**响应 200**

```json
{
  "status": "ready",
  "gpu": "RTX 6000 Ada Generation",
  "gpu_memory_used_gb": 15.23
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| status | string | `ready` / `busy` / `error` |
| gpu | string | GPU 型号 |
| gpu_memory_used_gb | number | 当前显存占用（GB） |

---

## 2. 检测接口（按模态分发）

ModelServer 对外提供三个端点，按文件模态分别调用：

| 模态 | 端点 |
|---|---|
| 图片 | `POST /api/detect/image` |
| 视频（含音频） | `POST /api/detect/video` |
| 音频 | `POST /api/detect/audio` |

三个端点的请求 / 响应**形态同构**，仅模型实现不同。

### 2.1 请求

**Headers**

```
Content-Type: application/json
```

**Body**

```json
{
  "oss_url": "https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/files/123/file_8e5b....mp4",
  "task_id": "task_345678abcdef...",
  "user_id": "123"
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| oss_url | string | 是 | OSS 上待检测文件的可访问 URL（公网地址或带签名地址）。ModelServer 必须能 GET 此 URL |
| task_id | string | 是 | dfkgo 侧任务 ID，用于日志追踪 + 拼接结果上传路径 |
| user_id | string | 是 | dfkgo 侧用户 ID，用于拼接结果上传路径 |

**ModelServer 行为约定**：
1. 收到请求后**主动 GET `oss_url`** 拉取源文件到本地临时目录
2. 运行对应模态模型完成检测，生成 mask / masked_image 本地临时文件
3. **将结果图上传到 OSS**：`PutObject` 到 `dfkgo-files/results/<user_id>/<task_id>/<原文件名>`
4. 响应中 `mask_urls` / `masked_image_urls` 等字段返回**上传后的 OSS URL**（完整 https 地址）
5. 检测完成后清理本地临时文件

**错误处理**：
- 拉取 oss_url 失败 → `502 BadGateway`
- OSS 写入 `results/` 失败 → `502 BadGateway`（`detail` 说明上传阶段失败）
- 文件过大 / 格式不支持 → `400 BadRequest`
- 部分结果图上传失败（例如 mask 成功、masked_image 失败）→ 响应 200，该字段返回空数组 / 空字符串

### 2.2 成功响应 200

```json
{
  "category": "tampered",
  "description": "The tampering involves the alteration of a single object within the image, specifically the snowboard, which is depicted in a different position and angle than the rest of the scene.",
  "raw_output": "... ASSISTANT: [CLS] [SEG] Type: The image is tampered ... [END]",
  "mask_urls": [
    "https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/results/123/task_345.../mask_0.jpg"
  ],
  "masked_image_urls": [
    "https://dfkgo-files.oss-cn-hangzhou.aliyuncs.com/results/123/task_345.../masked_img_0.jpg"
  ]
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| category | string | 是 | 检测分类，如 `real` / `tampered` / `fake` |
| description | string | 是 | 自然语言描述检测结论与依据 |
| raw_output | string | 否 | 模型原始输出，便于调试 |
| mask_urls | string[] | 否 | mask 图像的 **OSS URL**（已上传），通常 0~1 个 |
| masked_image_urls | string[] | 否 | 叠加 mask 后的可视化图像的 **OSS URL**（已上传），通常 0~1 个 |

**字段说明**：
- 所有图片 / 媒体 URL **必须是 OSS URL**（已由 ModelServer 上传到 `results/<user_id>/<task_id>/`），dfkgo 不做转存与 URL 改写
- 视频 / 音频模态可在响应中**新增**模态特有字段（如 `frame_anomalies`、`audio_segments`），同样遵守 OSS URL 约定；dfkgo 透传整个 JSON 到 `tasks.result_json`，不强制 schema 一致

### 2.3 错误响应

ModelServer 错误统一通过 HTTP 状态码 + JSON body 表达：

```json
{
  "detail": "Failed to fetch oss_url: 403 Forbidden"
}
```

| HTTP 状态 | 场景 |
|---|---|
| 400 | 请求体非法 / 文件格式不支持 / 文件过大 |
| 502 | 拉取 oss_url 失败（网络 / 权限 / 404） |
| 503 | GPU 忙 / 模型未就绪 |
| 500 | 内部错误（推理失败等） |

dfkgo 收到非 200 响应时：
- task 置 `failed`
- `error_message` 写入 `detail` 字段内容（最多 1024 字符）

---

## 3. ~~结果文件下载~~（废弃）

结果文件**不再由 ModelServer 提供下载服务**。上传到 OSS 后，客户端直接访问 `mask_urls` 中的 OSS URL。ModelServer 本地不保留结果文件（上传后立即清理）。

---

## 4. ModelServer 侧前置依赖

dfkgo 设计基于以下 ModelServer 改造前提（属 dfkgo Out of Scope）：

1. **OSS 访问权限**：ModelServer 拥有独立 RAM 账号，对 dfkgo 的 OSS Bucket：
   - `files/` 前缀：`oss:GetObject` / `oss:HeadObject` （读）
   - `results/` 前缀：`oss:PutObject` （写）
2. **三模态端点改造**：从原有 `/api/detect`（multipart）改造为 `/api/detect/{image|video|audio}`（JSON + oss_url + task_id + user_id）
3. **结果上传 OSS**：检测完成后 ModelServer 主动 `PutObject` 上传到 `dfkgo-files/results/<user_id>/<task_id>/`，响应中返回 OSS URL
4. **本地不保留**：上传后立即清理本地临时文件，ModelServer 不提供结果下载接口
5. （可选）**任务取消接口**：未来 ModelServer 提供取消接口后，dfkgo 软取消逻辑可升级为真取消

---

## 5. dfkgo 侧调用示例（Go）

```go
// service/task/modelclient.go
type DetectRequest struct {
    OssURL string `json:"oss_url"`
    TaskID string `json:"task_id"`
    UserID string `json:"user_id"`
}

func (c *HTTPModelClient) Detect(ctx context.Context, modality Modality, ossURL, taskID, userID string) ([]byte, error) {
    var endpoint string
    switch modality {
    case ModalityImage:
        endpoint = "/api/detect/image"
    case ModalityVideo:
        endpoint = "/api/detect/video"
    case ModalityAudio:
        endpoint = "/api/detect/audio"
    default:
        return nil, fmt.Errorf("unsupported modality: %s", modality)
    }

    body, _ := json.Marshal(DetectRequest{OssURL: ossURL, TaskID: taskID, UserID: userID})
    req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    raw, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("modelserver %d: %s", resp.StatusCode, string(raw))
    }
    return raw, nil  // 透传 JSON，写入 tasks.result_json
}
```
