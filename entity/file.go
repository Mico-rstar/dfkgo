package entity

type UploadInitRequest struct {
	FileName string `json:"fileName" binding:"required"`
	FileSize int64  `json:"fileSize" binding:"required"`
	MimeType string `json:"mimeType" binding:"required"`
	MD5      string `json:"md5" binding:"required"`
}

type UploadInitResponse struct {
	FileId string   `json:"fileId"`
	Hit    bool     `json:"hit"`
	OssUrl string   `json:"ossUrl,omitempty"`
	STS    *STSInfo `json:"sts,omitempty"`
}

type UploadCallbackRequest struct {
	FileId string `json:"fileId" binding:"required"`
}

type UploadCallbackResponse struct {
	FileId string `json:"fileId"`
	OssUrl string `json:"ossUrl"`
}

type STSInfo struct {
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SecurityToken   string `json:"securityToken"`
	Expiration      string `json:"expiration"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint"`
	ObjectKey       string `json:"objectKey"`
}
