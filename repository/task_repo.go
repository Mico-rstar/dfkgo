package repository

import (
	"database/sql"
	"dfkgo/model"
	"time"

	"gorm.io/gorm"
)

type TaskRepo struct {
	db *gorm.DB
}

func NewTaskRepo(db *gorm.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(task *model.Task) error {
	return r.db.Create(task).Error
}

func (r *TaskRepo) FindByTaskUID(taskUID string) (*model.Task, error) {
	var task model.Task
	err := r.db.Where("task_uid = ?", taskUID).First(&task).Error
	return &task, err
}

func (r *TaskRepo) FindByTaskUIDAndUserID(taskUID string, userID uint64) (*model.Task, error) {
	var task model.Task
	err := r.db.Where("task_uid = ? AND user_id = ?", taskUID, userID).First(&task).Error
	return &task, err
}

func (r *TaskRepo) UpdateStatus(id uint64, status string) error {
	return r.db.Model(&model.Task{}).Where("id = ?", id).Update("status", status).Error
}

func (r *TaskRepo) UpdateResult(id uint64, status string, resultJSON string, completedAt time.Time) error {
	return r.db.Model(&model.Task{}).Where("id = ?", id).Updates(map[string]any{
		"status":       status,
		"result_json":  resultJSON,
		"completed_at": sql.NullTime{Time: completedAt, Valid: true},
	}).Error
}

func (r *TaskRepo) UpdateFailed(id uint64, errorMessage string, completedAt time.Time) error {
	return r.db.Model(&model.Task{}).Where("id = ?", id).Updates(map[string]any{
		"status":        "failed",
		"error_message": errorMessage,
		"completed_at":  sql.NullTime{Time: completedAt, Valid: true},
	}).Error
}

func (r *TaskRepo) SetProcessing(id uint64, startedAt time.Time) error {
	return r.db.Model(&model.Task{}).Where("id = ?", id).Updates(map[string]any{
		"status":     "processing",
		"started_at": sql.NullTime{Time: startedAt, Valid: true},
	}).Error
}

func (r *TaskRepo) ListByUser(userID uint64, offset, limit int) ([]model.Task, int64, error) {
	var tasks []model.Task
	var total int64
	db := r.db.Where("user_id = ? AND deleted_at IS NULL", userID)
	db.Model(&model.Task{}).Count(&total)
	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, total, err
}

func (r *TaskRepo) SoftDelete(taskUID string, userID uint64) error {
	now := time.Now()
	return r.db.Model(&model.Task{}).Where("task_uid = ? AND user_id = ?", taskUID, userID).Update("deleted_at", &now).Error
}

func (r *TaskRepo) BatchSoftDelete(taskUIDs []string, userID uint64) (int64, error) {
	now := time.Now()
	result := r.db.Model(&model.Task{}).Where("task_uid IN ? AND user_id = ?", taskUIDs, userID).Update("deleted_at", &now)
	return result.RowsAffected, result.Error
}

func (r *TaskRepo) FindPendingTasks() ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.Where("status = ?", "pending").Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepo) FindProcessingTasks() ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.Where("status = ?", "processing").Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepo) FailProcessingTasks(errorMessage string) error {
	now := time.Now()
	return r.db.Model(&model.Task{}).Where("status = ?", "processing").Updates(map[string]any{
		"status":        "failed",
		"error_message": errorMessage,
		"completed_at":  sql.NullTime{Time: now, Valid: true},
	}).Error
}

func (r *TaskRepo) StatsForUser(userID uint64) (total int64, byModality map[string]int64, err error) {
	byModality = make(map[string]int64)
	db := r.db.Model(&model.Task{}).Where("user_id = ? AND deleted_at IS NULL AND status = ?", userID, "completed")
	db.Count(&total)

	type ModalityCount struct {
		Modality string
		Count    int64
	}
	var counts []ModalityCount
	r.db.Model(&model.Task{}).Select("modality, COUNT(*) as count").
		Where("user_id = ? AND deleted_at IS NULL AND status = ?", userID, "completed").
		Group("modality").Scan(&counts)
	for _, mc := range counts {
		byModality[mc.Modality] = mc.Count
	}
	return
}
