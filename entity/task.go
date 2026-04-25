package entity

type CreateTaskRequest struct {
	FileId   string `json:"fileId" binding:"required"`
	Modality string `json:"modality" binding:"required"`
}

type CreateTaskResponse struct {
	TaskId string `json:"taskId"`
	Status string `json:"status"`
}

type TaskStatusResponse struct {
	TaskId      string `json:"taskId"`
	Status      string `json:"status"`
	Modality    string `json:"modality"`
	CreatedAt   int64  `json:"createdAt"`
	StartedAt   *int64 `json:"startedAt"`
	CompletedAt *int64 `json:"completedAt"`
}

type TaskResultResponse struct {
	TaskId       string `json:"taskId"`
	Modality     string `json:"modality"`
	Status       string `json:"status"`
	Result       any    `json:"result"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	CompletedAt  *int64 `json:"completedAt"`
}
