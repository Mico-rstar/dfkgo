package handler

import (
	"dfkgo/api/response"
	"dfkgo/entity"
	"dfkgo/errcode"
	taskservice "dfkgo/service/task"
	"strconv"

	"github.com/gin-gonic/gin"
)

type HistoryHandler struct {
	taskService *taskservice.TaskService
}

func NewHistoryHandler(taskService *taskservice.TaskService) *HistoryHandler {
	return &HistoryHandler{taskService: taskService}
}

func (h *HistoryHandler) ListHistory(c *gin.Context) {
	userID := c.GetUint64("userID")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	historyItems, total, err := h.taskService.ListHistoryWithFiles(userID, page, limit)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}

	items := make([]entity.HistoryItem, 0, len(historyItems))
	for _, hi := range historyItems {
		item := entity.HistoryItem{
			TaskId:    hi.Task.TaskUID,
			Modality:  hi.Task.Modality,
			Status:    hi.Task.Status,
			CreatedAt: hi.Task.CreatedAt.Unix(),
			FileName:  hi.FileName,
			FileSize:  hi.FileSize,
		}
		if hi.Task.CompletedAt.Valid {
			ts := hi.Task.CompletedAt.Time.Unix()
			item.CompletedAt = &ts
		}
		items = append(items, item)
	}

	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}

	response.OK(c, entity.HistoryListResponse{
		Items: items,
		Pagination: entity.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
			Pages: pages,
		},
	})
}

func (h *HistoryHandler) DeleteHistory(c *gin.Context) {
	userID := c.GetUint64("userID")
	taskID := c.Param("taskId")

	if err := h.taskService.DeleteHistory(userID, taskID); err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OKWithMsg(c, "记录删除成功", nil)
}

func (h *HistoryHandler) BatchDeleteHistory(c *gin.Context) {
	userID := c.GetUint64("userID")
	var req entity.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, errcode.ErrInternal.Code, "参数错误: "+err.Error())
		return
	}

	count, err := h.taskService.BatchDeleteHistory(userID, req.TaskIds)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OKWithMsg(c, "批量删除成功", entity.BatchDeleteResponse{DeletedCount: count})
}

func (h *HistoryHandler) GetStats(c *gin.Context) {
	userID := c.GetUint64("userID")

	total, byModality, byCategory, err := h.taskService.GetStats(userID)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OK(c, entity.StatsResponse{
		Total:      total,
		ByModality: byModality,
		ByCategory: byCategory,
	})
}
