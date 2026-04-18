## Background context
Deepfake 后端项目，技术栈使用
- golang
- gin
- gorm + mysql
- aliyun oss（Go SDK V2 + STS 临时凭证）

## 设计文档
完整后端设计见 [docs/superpowers/specs/2026-04-18-dfkgo-backend-design.md](docs/superpowers/specs/2026-04-18-dfkgo-backend-design.md)。
本文件仅记录项目背景与高层范围，详细架构、API、表结构、决策记录请以设计文档为准。

## 基本逻辑
```
客户端/web <-> dfkgo <-> ModelServer
                 |-- oss --|
```
- 多模态数据由客户端 STS 直传 OSS，dfkgo 不经手字节流
- ModelServer 从 OSS 拉取数据进行检测后返回结果（dfkgo 仅传 oss_url + task_id）

## dfkgo scope
### In Scope（本期实现）
- 用户域：邮密注册（无邮箱验证）、邮密登录、Profile 查看与更新、头像 STS 上传
- 文件域：STS 临时凭证直传 OSS、上传完成回调、用户维度 MD5 秒传
- 任务域：统一 `/api/tasks` 接口，按模态（image/video/audio）分发到 ModelServer，状态/结果查询、软取消
- 历史域：分页查询、单条/批量软删、统计接口
- 基础设施：MySQL/GORM、OSS Go SDK V2 + STS、统一 envelope 响应、JWT 中间件、内存任务队列

### Out of Scope（明确推迟）
- 邮箱验证激活、密码修改、邮箱修改、令牌刷新、登出失效（不引入 Redis、不引入 SMTP）
- PDF 报告生成
- 多实例水平扩展、Redis/MQ 任务队列（架构上预留 `TaskQueue` 接口切换点）
- 任务真取消（依赖 ModelServer 提供取消接口）
- 历史记录多维筛选（仅分页）
- 跨模态（crossmodal）检测、进度百分比字段
- 前端能力、模型能力、模型服务生命周期

## 关键架构约定
- **API 路径前缀**：`/api`
- **响应格式**：统一 envelope `{code, message, data, timestamp}`，`code:0` 成功，非 0 为 6 位业务错误码
- **对外 ID**：`file_<uuid>` / `task_<uuid>`，DB 内部仍用自增 BIGINT
- **JWT**：HS256，24h 有效期，本期不实现失效机制
- **任务队列**：内存 buffered channel + worker pool（默认 4），启动时扫库恢复孤儿任务
- **ModelServer 调用**：HTTP JSON `{oss_url, task_id}`，按 modality 路由到 `/api/detect/{image|video|audio}`
- **数据表**：users / files / tasks 三张表（详见设计文档第 4 节）
- **OSS 上传**：STS 临时凭证模式（前端已实现，沿用），dfkgo 配 RAM Role + 长期 AK/SK

## 前置外部依赖
- ModelServer 接口改造：提供按模态分类的 3 个端点，接收 `{oss_url, task_id}` JSON
- ModelServer 具备目标 OSS Bucket 只读权限

## API 文档
### dfkgo ↔ 前端/客户端
[docs/api.md]

### dfkgo ↔ ModelServer
[docs/model_server_api.md]

### OSS Go SDK 用法
[docs/aliyun_oss_go_sdk.md]

## hook
每轮回答前加入前缀 **，以告知用户你已读取此文档并充分理解
