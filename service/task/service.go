package task

import (
	"context"
	"dfkgo/errcode"
	"dfkgo/model"
	"dfkgo/repository"
	"encoding/json"
	"log"
	"strings"

	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo *repository.TaskRepo
	fileRepo *repository.FileRepo
	queue    TaskQueue
}

func NewTaskService(taskRepo *repository.TaskRepo, fileRepo *repository.FileRepo, queue TaskQueue) *TaskService {
	return &TaskService{taskRepo: taskRepo, fileRepo: fileRepo, queue: queue}
}

func (s *TaskService) CreateTask(userID uint64, fileID string, modality string) (taskUID string, err error) {
	// 1. 校验 file 存在且属于当前用户且 upload_status=completed
	file, err := s.fileRepo.FindByFileUID(fileID)
	if err != nil {
		return "", errcode.ErrFileIDNotFound
	}
	if file.UserID != userID {
		return "", errcode.ErrFileIDNotFound
	}
	if file.UploadStatus != "completed" {
		return "", errcode.ErrFileIDNotFound
	}

	// 2. 校验 modality
	modality = strings.ToLower(modality)
	if modality != "image" && modality != "video" && modality != "audio" {
		return "", errcode.ErrModalityMismatch
	}

	// 3. 生成 taskUID
	taskUID = "task_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// 4. 创建 task 记录
	task := &model.Task{
		TaskUID:  taskUID,
		UserID:   userID,
		FileID:   file.ID,
		Modality: modality,
		Status:   "pending",
	}
	if err := s.taskRepo.Create(task); err != nil {
		return "", errcode.ErrDBError
	}

	// 5. Push 入队
	if err := s.queue.Push(context.Background(), taskUID); err != nil {
		return "", errcode.ErrInternal
	}

	return taskUID, nil
}

func (s *TaskService) GetTaskStatus(userID uint64, taskUID string) (*model.Task, error) {
	task, err := s.taskRepo.FindByTaskUIDAndUserID(taskUID, userID)
	if err != nil {
		return nil, errcode.ErrTaskNotFound
	}
	return task, nil
}

func (s *TaskService) GetTaskResult(userID uint64, taskUID string) (*model.Task, error) {
	task, err := s.taskRepo.FindByTaskUIDAndUserID(taskUID, userID)
	if err != nil {
		return nil, errcode.ErrTaskNotFound
	}
	if task.Status != "completed" && task.Status != "failed" {
		return nil, errcode.ErrTaskNotCompleted
	}
	return task, nil
}

func (s *TaskService) CancelTask(userID uint64, taskUID string) error {
	task, err := s.taskRepo.FindByTaskUIDAndUserID(taskUID, userID)
	if err != nil {
		return errcode.ErrTaskNotFound
	}
	if task.Status != "pending" && task.Status != "processing" {
		return errcode.ErrTaskCannotCancel
	}
	return s.taskRepo.UpdateStatus(task.ID, "cancelled")
}

func (s *TaskService) RecoverOrphanTasks() error {
	// 1. processing -> failed
	if err := s.taskRepo.FailProcessingTasks("service restarted"); err != nil {
		return err
	}
	// 2. pending -> 重新入队
	pendingTasks, err := s.taskRepo.FindPendingTasks()
	if err != nil {
		return err
	}
	for _, t := range pendingTasks {
		if err := s.queue.Push(context.Background(), t.TaskUID); err != nil {
			log.Printf("[RecoverOrphanTasks] push %s failed: %v", t.TaskUID, err)
		}
	}
	log.Printf("[RecoverOrphanTasks] recovered %d pending tasks", len(pendingTasks))
	return nil
}

type HistoryItemData struct {
	Task     model.Task
	FileName string
	FileSize int64
}

// ListHistoryWithFiles 分页查询历史并批量关联文件信息
func (s *TaskService) ListHistoryWithFiles(userID uint64, page, limit int) ([]HistoryItemData, int64, error) {
	offset := (page - 1) * limit
	tasks, total, err := s.taskRepo.ListByUser(userID, offset, limit)
	if err != nil {
		return nil, 0, errcode.ErrDBError
	}

	fileIDs := make([]uint64, 0, len(tasks))
	for _, t := range tasks {
		fileIDs = append(fileIDs, t.FileID)
	}
	files, _ := s.fileRepo.FindByIDs(fileIDs)
	fileMap := make(map[uint64]model.File, len(files))
	for _, f := range files {
		fileMap[f.ID] = f
	}

	items := make([]HistoryItemData, 0, len(tasks))
	for _, t := range tasks {
		item := HistoryItemData{Task: t}
		if f, ok := fileMap[t.FileID]; ok {
			item.FileName = f.FileName
			item.FileSize = f.FileSize
		}
		items = append(items, item)
	}
	return items, total, nil
}

// DeleteHistory 单条软删
func (s *TaskService) DeleteHistory(userID uint64, taskUID string) error {
	return s.taskRepo.SoftDelete(taskUID, userID)
}

// BatchDeleteHistory 批量软删
func (s *TaskService) BatchDeleteHistory(userID uint64, taskUIDs []string) (int64, error) {
	return s.taskRepo.BatchSoftDelete(taskUIDs, userID)
}

// GetStats 统计
func (s *TaskService) GetStats(userID uint64) (total int64, byModality map[string]int64, byCategory map[string]int64, err error) {
	total, byModality, err = s.taskRepo.StatsForUser(userID)
	if err != nil {
		return 0, nil, nil, errcode.ErrDBError
	}
	byCategory = make(map[string]int64)
	results, err := s.taskRepo.FindCompletedResultsForUser(userID)
	if err != nil {
		return total, byModality, byCategory, nil
	}
	for _, raw := range results {
		var result map[string]any
		if err := json.Unmarshal([]byte(raw), &result); err == nil {
			if cat, ok := result["category"].(string); ok {
				byCategory[cat]++
			}
		}
	}
	return total, byModality, byCategory, nil
}
