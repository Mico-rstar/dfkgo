# 阿里云 OSS Go SDK V2 用法速查

> 本文档基于阿里云官方帮助中心 [对象存储 OSS - 开发参考](https://help.aliyun.com/zh/oss/developer-reference/manual-for-go-sdk-v2/) 整理，覆盖 dfkgo 项目所需的初始化、上传（简单 / 分片 / 上传管理器 / 预签名）、下载、对象管理等场景。SDK 模块路径为 `github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss`，**与 V1 (`github.com/aliyun/aliyun-oss-go-sdk/oss`) 不兼容**，新项目请直接使用 V2。

参考来源：
- 手册总览：https://help.aliyun.com/zh/oss/developer-reference/manual-for-go-sdk-v2/
- 凭证管理：https://help.aliyun.com/zh/sdk/developer-reference/v2-manage-go-access-credentials
- 简单上传：https://help.aliyun.com/zh/oss/developer-reference/v2-simple-upload
- 分片上传：https://help.aliyun.com/zh/oss/developer-reference/v2-multipart-upload
- 上传管理器：https://help.aliyun.com/zh/oss/developer-reference/v2-uploader
- 预签名上传：https://help.aliyun.com/zh/oss/developer-reference/v2-presign-upload
- 简单下载：https://help.aliyun.com/zh/oss/developer-reference/v2-download-objects-as-files
- 预签名下载：https://help.aliyun.com/zh/oss/developer-reference/v2-presign-download
- 删除对象：https://help.aliyun.com/zh/oss/developer-reference/v2-delete-objects
- 判断对象存在：https://help.aliyun.com/zh/oss/developer-reference/v2-determine-whether-an-object-exists
- GitHub 仓库：https://github.com/aliyun/alibabacloud-oss-go-sdk-v2

---

## 1. 环境与安装

- Go 版本要求：**1.18 及以上**。
- 安装：

```bash
go get github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss
```

- 引入：

```go
import (
    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)
```

---

## 2. 凭证配置

推荐通过环境变量管理 AK/SK，避免硬编码：

```bash
export OSS_ACCESS_KEY_ID="<your_access_key_id>"
export OSS_ACCESS_KEY_SECRET="<your_access_key_secret>"
# 若使用 STS 临时凭证，再额外导出
export OSS_SESSION_TOKEN="<sts_security_token>"
```

常见 `CredentialsProvider`：

| Provider | 用途 |
| --- | --- |
| `credentials.NewEnvironmentVariableCredentialsProvider()` | 从 `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET` / `OSS_SESSION_TOKEN` 加载 |
| `credentials.NewStaticCredentialsProvider(ak, sk, token)` | 直接传入静态 AK/SK（仅本地调试） |
| 默认凭据链 | 顺序读取环境变量、配置文件、ECS RAM 角色等 |

---

## 3. 客户端初始化

```go
cfg := oss.LoadDefaultConfig().
    WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
    WithRegion("cn-hangzhou") // 例如华东1

client := oss.NewClient(cfg)
```

要点：
- `WithRegion` 必填；同地域内网访问可改用 `WithEndpoint("oss-cn-hangzhou-internal.aliyuncs.com")`。
- 默认使用 V4 签名（`OSS4-HMAC-SHA256`）。
- `client` 协程安全，应作为单例复用，**不要每次请求都新建**。

dfkgo 项目建议在 `service/oss` 下封装一个 `provider`：

```go
type OssClient struct {
    Client *oss.Client
    Bucket string
}

func New(region, bucket string) *OssClient {
    cfg := oss.LoadDefaultConfig().
        WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
        WithRegion(region)
    return &OssClient{Client: oss.NewClient(cfg), Bucket: bucket}
}
```

---

## 4. 简单上传 (PutObject)

适用：≤ 5 GB 的小文件 / 业务可一次性放入内存或文件流的对象。

权限：调用方需要 `oss:PutObject`。

### 4.1 上传字符串 / 字节流

```go
body := strings.NewReader("hi oss")

result, err := client.PutObject(ctx, &oss.PutObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectName),
    Body:   body,
})
```

`bytes.NewReader([]byte{...})` 同理。

### 4.2 上传本地文件（带进度回调）

```go
result, err := client.PutObjectFromFile(ctx,
    &oss.PutObjectRequest{
        Bucket: oss.Ptr(bucketName),
        Key:    oss.Ptr(objectName),
        ProgressFn: func(increment, transferred, total int64) {
            log.Printf("transferred=%d total=%d", transferred, total)
        },
    },
    "/local/path/exampleobject.txt",
)
```

### 4.3 上传后回调（Callback）

将回调描述对象 base64 编码后挂到 `Callback` 字段，OSS 完成上传时会回调业务 URL：

```go
cbMap := map[string]string{
    "callbackUrl":      "https://api.example.com/oss/callback",
    "callbackBody":     "bucket=${bucket}&object=${object}&size=${size}&taskId=${x:taskId}",
    "callbackBodyType": "application/x-www-form-urlencoded",
}
cbJSON, _ := json.Marshal(cbMap)

req := &oss.PutObjectRequest{
    Bucket:   oss.Ptr(bucketName),
    Key:      oss.Ptr(objectName),
    Body:     body,
    Callback: oss.Ptr(base64.StdEncoding.EncodeToString(cbJSON)),
}
```

dfkgo 可借此让 OSS 在上传成功后通知后端创建检测任务，避免客户端"先上传后请求"两步握手。

---

## 5. 分片上传 (Multipart Upload)

适用：单文件 > 100 MB、需要断点续传或并发加速、不稳定网络。流程：

1. `InitiateMultipartUpload` → 拿到全局唯一 `UploadId`
2. 并发 `UploadPart`（每个分片记录 `PartNumber + ETag`）
3. `CompleteMultipartUpload`（按 PartNumber 顺序合并）；中途可调用 `AbortMultipartUpload` 取消

```go
// 1) 初始化
init, err := client.InitiateMultipartUpload(ctx, &oss.InitiateMultipartUploadRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectName),
})
uploadId := *init.UploadId

// 2) 并发上传分片
var (
    parts []oss.UploadPart
    mu    sync.Mutex
    wg    sync.WaitGroup
)
for i, chunk := range chunks { // chunks 为 io.Reader 切片
    wg.Add(1)
    partNumber := int32(i + 1)
    go func(pn int32, body io.Reader) {
        defer wg.Done()
        r, err := client.UploadPart(ctx, &oss.UploadPartRequest{
            Bucket:     oss.Ptr(bucketName),
            Key:        oss.Ptr(objectName),
            PartNumber: pn,
            UploadId:   oss.Ptr(uploadId),
            Body:       body,
        })
        if err != nil { /* handle */ }
        mu.Lock()
        parts = append(parts, oss.UploadPart{PartNumber: pn, ETag: r.ETag})
        mu.Unlock()
    }(partNumber, chunk)
}
wg.Wait()

// 3) 完成
sort.Slice(parts, func(i, j int) bool { return parts[i].PartNumber < parts[j].PartNumber })
_, err = client.CompleteMultipartUpload(ctx, &oss.CompleteMultipartUploadRequest{
    Bucket:                  oss.Ptr(bucketName),
    Key:                     oss.Ptr(objectName),
    UploadId:                oss.Ptr(uploadId),
    CompleteMultipartUpload: &oss.CompleteMultipartUpload{Parts: parts},
})
```

注意：
- **同一 `PartNumber` 重新上传会覆盖原数据**，可用于失败重传。
- OSS 通过分片 ETag 中的 MD5 校验数据完整性，写入不一致会返回 `InvalidDigest`。
- 长时间未完成的 UploadId 占用空间，应定时通过 `ListMultipartUploads` 清理。

---

## 6. 上传管理器 (Uploader) —— 推荐用于大文件

`Uploader` 是 SDK 对分片上传的高阶封装，自动并发、自动断点续传、可上传本地文件或任意 `io.Reader`，dfkgo 处理 ≤500MB 的音视频文件直接用它即可。

### 6.1 基础用法（上传本地文件）

```go
u := client.NewUploader()
result, err := u.UploadFile(ctx,
    &oss.PutObjectRequest{
        Bucket: oss.Ptr(bucketName),
        Key:    oss.Ptr(objectName),
    },
    "/local/path/yourFileName",
)
```

### 6.2 启用断点续传

```go
u := client.NewUploader(func(uo *oss.UploaderOptions) {
    uo.CheckpointDir   = "/var/lib/dfkgo/oss-checkpoints/"
    uo.EnableCheckpoint = true
})
```

`CheckpointDir` 是必备的本地目录，SDK 会写入元数据描述已完成的分片；进程崩溃后再次调用会从断点恢复。dfkgo 服务端代理上传时建议使用专门挂载的临时目录。

### 6.3 上传 io.Reader（流式）

```go
file, _ := os.Open(localPath)
defer file.Close()

result, err := u.UploadFrom(ctx,
    &oss.PutObjectRequest{Bucket: oss.Ptr(bucketName), Key: oss.Ptr(objectName)},
    file,
)
```

可调参数（`UploaderOptions`）：
- `PartSize`：分片大小，默认 6 MiB。
- `ParallelNum`：并发分片数，默认 5。
- `LeavePartsOnError`：失败时是否保留已上传分片（断点续传必开）。

---

## 7. 预签名 URL（直传 / 直下）

服务端签发，客户端直接走 OSS，**不经后端转发**，dfkgo 处理大文件强烈推荐此方式。

### 7.1 PUT 预签名（前端直传）

```go
res, err := client.Presign(ctx, &oss.PutObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectName),
}, oss.PresignExpires(10*time.Minute))

// 返回字段
// res.Method        -> "PUT"
// res.URL           -> 给前端的预签名地址
// res.Expiration    -> URL 过期时间
// res.SignedHeaders -> 若签名包含 Content-Type 等头，前端 PUT 时必须带上
```

⚠️ 若签名时指定了 `Content-Type` / `Content-MD5` / 自定义 `x-oss-meta-*`，调用方 PUT 时必须使用一致的请求头，否则签名不通过。

客户端示例：

```bash
curl -X PUT -T ./local.mp4 \
  -H "Content-Type: video/mp4" \
  "<res.URL>"
```

### 7.2 GET 预签名（前端直下 / 临时分享）

```go
res, err := client.Presign(ctx, &oss.GetObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectName),
}, oss.PresignExpires(10*time.Minute))
// res.URL 即下载链接
```

---

## 8. 下载 (GetObject)

### 8.1 下载到本地文件

```go
_, err := client.GetObjectToFile(ctx, &oss.GetObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectName),
}, "/local/path/download.file")
```

支持条件下载：`IfModifiedSince`, `IfUnmodifiedSince`, `IfMatch`, `IfNoneMatch`。

### 8.2 下载到内存 / 流式读取

```go
out, err := client.GetObject(ctx, &oss.GetObjectRequest{
    Bucket: oss.Ptr(bucketName), Key: oss.Ptr(objectName),
})
defer out.Body.Close()
data, _ := io.ReadAll(out.Body)
```

亦可使用 `client.NewDownloader()` 实现并发分片下载与断点续传，API 风格与 Uploader 对称。

---

## 9. 对象管理常用 API

| 操作 | 方法 | 说明 |
| --- | --- | --- |
| 判断对象是否存在 | `client.IsObjectExist(ctx, bucket, key)` | 返回 `bool, error` |
| 删除单个对象 | `client.DeleteObject(ctx, &oss.DeleteObjectRequest{...})` | 需 `oss:DeleteObject` |
| 批量删除（最多 1000） | `client.DeleteMultipleObjects(ctx, &oss.DeleteMultipleObjectsRequest{...})` | 服务端原子操作 |
| 列举对象 | `client.NewListObjectsV2Paginator(&oss.ListObjectsV2Request{...})` | 返回分页器 `HasNext / NextPage` |
| 拷贝对象 | `client.CopyObject(...)` / 大文件用 `client.NewCopier()` | 同 / 跨 Bucket 复制 |
| 修改元数据 / 存储类型 | `CopyObject` 同源覆盖即可 |  |
| 获取对象元信息 | `client.HeadObject(...)` | 仅返回 header，开销低 |

示例：删除单文件

```go
_, err := client.DeleteObject(ctx, &oss.DeleteObjectRequest{
    Bucket: oss.Ptr(bucketName),
    Key:    oss.Ptr(objectName),
})
```

---

## 10. 在 dfkgo 项目中的落地建议

结合 VectorX.md 中"客户端 ↔ dfkgo ↔ ModelServer，多模态文件经 OSS 解耦"的设计：

1. **凭证**：把 `region` / `bucket` / `oss_endpoint`(可选) 写入 `app.env`，AK/SK 通过 `OSS_ACCESS_KEY_ID` / `OSS_ACCESS_KEY_SECRET` 注入容器；线上推荐 RAM 角色或 STS 临时凭证。
2. **客户端单例**：`service/oss` 包内通过 `sync.Once` 创建一个 `*oss.Client`，与 `config`、`api.Server` 风格保持一致。
3. **上传方案**：
   - ≤ 50 MB 的图片 / 头像：走后端 `POST /api/upload`，服务端直接 `PutObject`，可同时设置 Callback 通知任务系统。
   - > 50 MB 或音视频：服务端签发 `Presign(PUT)`，前端直传 OSS，再回调 `POST /api/tasks` 创建检测任务，避免占用 dfkgo 带宽与连接。
   - 服务端如确需代为上传大文件，统一使用 `client.NewUploader()` 并开启断点续传。
4. **下载 / 报告**：检测结果中的预览音视频、PDF 报告统一用 `Presign(GET)` 签发短期可访问链接，TTL 与业务页面停留时长匹配（如 10 ~ 30 分钟）。
5. **生命周期**：
   - 在 OSS 控制台为 `tmp/`、`reports/` 前缀配置生命周期规则（如临时上传 7 天后自动删除），降低成本；
   - 业务侧在删除任务历史时调用 `DeleteMultipleObjects` 清理对应对象。
6. **错误处理**：捕获 `*oss.ServiceError`，关注 `Code` / `RequestId` 字段并落 metrics；签名错误优先排查 region、Endpoint、签名头一致性。
7. **本地开发**：可在 `app.env` 中通过 `OSS_ENDPOINT` 指向 [LocalStack S3](https://docs.localstack.cloud) 或 OSS 兼容 mock，减少联调成本（SDK 兼容自定义 endpoint）。

---

## 11. 参考资料

- 手册总览：https://help.aliyun.com/zh/oss/developer-reference/manual-for-go-sdk-v2/
- V1 → V2 迁移指南：https://help.aliyun.com/zh/oss/developer-reference/migration-guide-in-go
- GitHub 仓库 + 示例代码：https://github.com/aliyun/alibabacloud-oss-go-sdk-v2 （`sample/` 目录）
- 凭证（含 STS / OIDC / URI / Bearer）：https://help.aliyun.com/zh/sdk/developer-reference/v2-manage-go-access-credentials
