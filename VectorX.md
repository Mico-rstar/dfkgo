## Background context
Deepfake 后端项目，技术栈使用
- golang
- gin
- gorm + mysql
- aliyun oss

## PR

后端服务（dfkgo）需对外提供以下能力，所有受保护接口需 JWT Bearer Token 鉴权。

### 1. 用户认证（注册 / 登录）
- `POST /api/auth/register`：邮箱 + 密码注册。
  - 邮箱唯一性校验、密码加密存储。
  - 生成邮件验证令牌并发送验证邮件。
  - 提供验证令牌校验接口完成激活。
- `POST /api/auth/login`：邮箱 + 密码登录，校验通过后签发 JWT（有效期 24h）。
- 提供令牌刷新接口与登出接口。

### 2. 文件上传
- `POST /api/upload`：multipart 上传待检测音视频文件。
  - 类型白名单：mp4 / mov / avi / mp3 / wav。
  - 大小限制：≤ 500MB。
  - 支持分块上传（断点续传）。
  - 临时文件存储管理，返回唯一任务 ID。

### 3. Deepfake 多模型检测
- 任务队列管理：接收检测任务并调度执行。
- 并行调用四个模型（视频、音频、图像、跨模态）至 ModelServer。
- 任务进度与状态跟踪接口（排队 / 进行中 / 完成 / 失败）。
- 结果聚合算法：将各模型结果合成一个综合概率。
- 任务取消接口与超时处理机制。

### 4. 检测结果
- 提供结构化结果数据接口（综合结果 + 各模型明细）。
- 报告数据聚合，提供 PDF 报告生成与下载服务。
- 结果缓存机制，避免重复计算。

### 5. 历史记录
- 历史记录查询接口（分页）。
- 支持按时间检索。
- 记录软删除（单条 / 批量）。
- 数据统计接口。

### 6. 用户个人信息
- `GET /api/user/profile`：返回 username、email、nickname、avatarUrl、bio。需 JWT。
- `PUT /api/user/profile`：更新昵称、简介等基本资料（头像通过单独接口）。需 JWT。
- `POST /api/user/upload-avatar`：multipart 上传头像，保存到对象存储/静态目录，更新 avatarUrl 字段并返回新 URL。
- `PUT /api/user/change-password`：校验旧密码，加密存储新密码；成功后可使该用户其他 JWT 失效。需 JWT。
- `POST /api/user/change-email`：检查新邮箱是否被占用，生成验证码存入 Redis（有效期 5 分钟）并发送到新邮箱。
- `PUT /api/user/confirm-new-email`：核对验证码后更新邮箱字段，并向旧邮箱发送通知邮件。需 JWT。



## 基本逻辑
客户端/web <-> dfkgo <-> ModelServer
                ｜-- oss --｜
- 多模态数据上传到oss
- ModelServer从oss拉取数据进行检测后返回结果

## dfkgo scope
### In scope
- 用户鉴权
- 用户相关状态维护
- 检测历史、检测结果状态维护

### Out of scope
- 前端能力
- 模型能力
- 模型服务生命周期，将模型服务视为一个常驻服务
- 多模态数据数据持久化，解耦到oss

## API文档
### dfkgo-前端/客户端
[docs/api.md]

### dfkgo-ModelServer
[docs/model_server_api.md]

