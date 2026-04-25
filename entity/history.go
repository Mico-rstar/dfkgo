package entity

type HistoryItem struct {
	TaskId      string `json:"taskId"`
	FileName    string `json:"fileName"`
	Modality    string `json:"modality"`
	FileSize    int64  `json:"fileSize"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"createdAt"`
	CompletedAt *int64 `json:"completedAt"`
}

type Pagination struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
	Pages int   `json:"pages"`
}

type HistoryListResponse struct {
	Items      []HistoryItem `json:"items"`
	Pagination Pagination    `json:"pagination"`
}

type BatchDeleteRequest struct {
	TaskIds []string `json:"taskIds" binding:"required"`
}

type BatchDeleteResponse struct {
	DeletedCount int64 `json:"deletedCount"`
}

type StatsResponse struct {
	Total      int64            `json:"total"`
	ByModality map[string]int64 `json:"byModality"`
	ByCategory map[string]int64 `json:"byCategory"`
}
