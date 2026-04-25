package handler

import (
	"dfkgo/api/response"
	"dfkgo/entity"
	"dfkgo/errcode"
	"encoding/json"
	taskservice "dfkgo/service/task"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskService *taskservice.TaskService
}

func NewTaskHandler(taskService *taskservice.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID := c.GetUint64("userID")
	var req entity.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrInternal.Code, "参数错误: "+err.Error())
		return
	}

	taskUID, err := h.taskService.CreateTask(userID, req.FileId, req.Modality)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}

	response.OK(c, entity.CreateTaskResponse{
		TaskId: taskUID,
		Status: "pending",
	})
}

func (h *TaskHandler) GetTaskStatus(c *gin.Context) {
	userID := c.GetUint64("userID")
	taskID := c.Param("taskId")

	task, err := h.taskService.GetTaskStatus(userID, taskID)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}

	resp := entity.TaskStatusResponse{
		TaskId:    task.TaskUID,
		Status:    task.Status,
		Modality:  task.Modality,
		CreatedAt: task.CreatedAt.Unix(),
	}
	if task.StartedAt.Valid {
		t := task.StartedAt.Time.Unix()
		resp.StartedAt = &t
	}
	if task.CompletedAt.Valid {
		t := task.CompletedAt.Time.Unix()
		resp.CompletedAt = &t
	}
	response.OK(c, resp)
}

func (h *TaskHandler) GetTaskResult(c *gin.Context) {
	userID := c.GetUint64("userID")
	taskID := c.Param("taskId")

	task, err := h.taskService.GetTaskResult(userID, taskID)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}

	resp := entity.TaskResultResponse{
		TaskId:       task.TaskUID,
		Modality:     task.Modality,
		Status:       task.Status,
		ErrorMessage: task.ErrorMessage,
	}
	if task.CompletedAt.Valid {
		t := task.CompletedAt.Time.Unix()
		resp.CompletedAt = &t
	}
	if task.ResultJSON != nil {
		var result any
		if err := json.Unmarshal([]byte(*task.ResultJSON), &result); err == nil {
			resp.Result = result
		}
	}
	response.OK(c, resp)
}

func (h *TaskHandler) CancelTask(c *gin.Context) {
	userID := c.GetUint64("userID")
	taskID := c.Param("taskId")

	if err := h.taskService.CancelTask(userID, taskID); err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OKWithMsg(c, "任务已取消", nil)
}
