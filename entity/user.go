package entity

type ProfileResponse struct {
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	AvatarUrl string `json:"avatarUrl"`
	Phone     string `json:"phone"`
}

type UpdateProfileRequest struct {
	Nickname *string `json:"nickname"`
	Phone    *string `json:"phone"`
}

type AvatarUploadInitRequest struct {
	MimeType string `json:"mimeType" binding:"required"`
	FileSize int64  `json:"fileSize" binding:"required"`
}

type AvatarUploadInitResponse struct {
	AccessKeyID     string   `json:"accessKeyId"`
	AccessKeySecret string   `json:"accessKeySecret"`
	SecurityToken   string   `json:"securityToken"`
	Expiration      string   `json:"expiration"`
	Bucket          string   `json:"bucket"`
	Region          string   `json:"region"`
	Endpoint        string   `json:"endpoint"`
	ObjectKey       string   `json:"objectKey"`
	MaxFileSize     int64    `json:"maxFileSize"`
	AllowedFileTypes []string `json:"allowedFileTypes"`
}

type AvatarUploadCallbackRequest struct {
	ObjectKey string `json:"objectKey" binding:"required"`
}

type AvatarUploadCallbackResponse struct {
	AvatarUrl string `json:"avatarUrl"`
}

type FetchAvatarResponse struct {
	AvatarUrl string `json:"avatarUrl"`
}
