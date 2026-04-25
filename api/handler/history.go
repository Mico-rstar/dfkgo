package handler

import (
	"dfkgo/api/response"
	"dfkgo/entity"
	"dfkgo/errcode"
	"dfkgo/repository"
	taskservice "dfkgo/service/task"
	"strconv"

	"github.com/gin-gonic/gin"
)

type HistoryHandler struct {
	taskService *taskservice.TaskService
	fileRepo    *repository.FileRepo
}

func NewHistoryHandler(taskService *taskservice.TaskService, fileRepo *repository.FileRepo) *HistoryHandler {
	return &HistoryHandler{taskService: taskService, fileRepo: fileRepo}
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

	tasks, total, err := h.taskService.ListHistory(userID, page, limit)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}

	items := make([]entity.HistoryItem, 0, len(tasks))
	for _, t := range tasks {
		item := entity.HistoryItem{
			TaskId:    t.TaskUID,
			Modality:  t.Modality,
			Status:    t.Status,
			CreatedAt: t.CreatedAt.Unix(),
		}
		if t.CompletedAt.Valid {
			ts := t.CompletedAt.Time.Unix()
			item.CompletedAt = &ts
		}
		// 查文件信息
		file, err := h.fileRepo.FindByID(t.FileID)
		if err == nil {
			item.FileName = file.FileName
			item.FileSize = file.FileSize
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
