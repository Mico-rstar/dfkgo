package errcode

// 认证/授权 4011xx
var (
	ErrInvalidEmail    = New(401101, "邮箱格式不合法")
	ErrInvalidPassword = New(401102, "密码不符合规则")
	ErrUserNotFound    = New(401004, "用户不存在")
	ErrWrongPassword   = New(401005, "用户密码错误，请重新输入")
	ErrUserExists      = New(401009, "用户已存在")
	ErrUnauthorized    = New(401001, "未授权访问")
	ErrTokenExpired    = New(401002, "登录已过期")
)

// 用户域 4012xx
var (
	ErrAvatarNotFound    = New(401201, "头像不存在")
	ErrAvatarOSSNotFound = New(401202, "文件未在 OSS 上找到")
	ErrInvalidMimeType   = New(401203, "不支持的文件类型")
	ErrFileTooLarge      = New(401204, "文件大小超出限制")
)

// 文件域 4013xx
var (
	ErrFileTypeNotSupported = New(401301, "文件类型不支持")
	ErrFileSizeExceeded     = New(401302, "文件大小超出限制")
	ErrFileOSSNotFound      = New(401303, "文件未在 OSS 上找到")
	ErrFileIDNotFound       = New(401304, "fileId 不存在")
	ErrFileExists           = New(401018, "文件已存在") // 历史兼容码
)

// 任务域 4014xx
var (
	ErrModalityMismatch = New(401401, "modality 与文件类型不匹配")
	ErrTaskNotFound     = New(401402, "任务不存在")
	ErrTaskCannotCancel = New(401403, "任务状态不允许取消")
	ErrTaskNotCompleted = New(401404, "任务尚未完成")
)

// 内部错误 5000xx
var (
	ErrInternal    = New(500001, "内部服务错误")
	ErrDBError     = New(500002, "数据库异常")
	ErrOSSError    = New(500003, "OSS 服务异常")
	ErrModelServer = New(500004, "ModelServer 服务异常")
)
