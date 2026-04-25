package repository

import (
	"dfkgo/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTaskTestDB(t *testing.T) (*gorm.DB, *UserRepo, *FileRepo, *TaskRepo) {
	db, err := InitTestDB()
	require.NoError(t, err)
	return db, NewUserRepo(db), NewFileRepo(db), NewTaskRepo(db)
}

func createTestUserAndFile(t *testing.T, userRepo *UserRepo, fileRepo *FileRepo) (*model.User, *model.File) {
	user := &model.User{
		Email:        "tasktest@example.com",
		PasswordHash: "$2a$10$fakehash",
	}
	err := userRepo.Create(user)
	require.NoError(t, err)

	file := &model.File{
		FileUID:      "file_fortask",
		UserID:       user.ID,
		FileName:     "test.jpg",
		MimeType:     "image/jpeg",
		FileSize:     1024,
		Modality:     "image",
		MD5:          "d41d8cd98f00b204e9800998ecf8427e",
		OSSBucket:    "dfkgo-files",
		OSSObjectKey: "files/1/file_fortask.jpg",
		OSSURL:       "https://dfkgo-files.oss.aliyuncs.com/files/1/file_fortask.jpg",
		UploadStatus: "completed",
	}
	err = fileRepo.Create(file)
	require.NoError(t, err)
	return user, file
}

func TestTaskRepo_CreateAndFind(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	task := &model.Task{
		TaskUID:  "task_test001",
		UserID:   user.ID,
		FileID:   file.ID,
		Modality: "image",
		Status:   "pending",
	}
	err := taskRepo.Create(task)
	assert.NoError(t, err)
	assert.NotZero(t, task.ID)

	found, err := taskRepo.FindByTaskUID("task_test001")
	assert.NoError(t, err)
	assert.Equal(t, "pending", found.Status)

	found, err = taskRepo.FindByTaskUIDAndUserID("task_test001", user.ID)
	assert.NoError(t, err)
	assert.Equal(t, task.ID, found.ID)
}

func TestTaskRepo_UpdateStatus(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	task := &model.Task{
		TaskUID:  "task_status",
		UserID:   user.ID,
		FileID:   file.ID,
		Modality: "image",
		Status:   "pending",
	}
	err := taskRepo.Create(task)
	require.NoError(t, err)

	err = taskRepo.UpdateStatus(task.ID, "processing")
	assert.NoError(t, err)

	found, err := taskRepo.FindByTaskUID("task_status")
	assert.NoError(t, err)
	assert.Equal(t, "processing", found.Status)
}

func TestTaskRepo_UpdateResult(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	task := &model.Task{
		TaskUID:  "task_result",
		UserID:   user.ID,
		FileID:   file.ID,
		Modality: "image",
		Status:   "processing",
	}
	err := taskRepo.Create(task)
	require.NoError(t, err)

	resultJSON := `{"score": 0.95}`
	err = taskRepo.UpdateResult(task.ID, "completed", resultJSON, task.CreatedAt)
	assert.NoError(t, err)

	found, err := taskRepo.FindByTaskUID("task_result")
	assert.NoError(t, err)
	assert.Equal(t, "completed", found.Status)
	assert.NotNil(t, found.ResultJSON)
	assert.Equal(t, resultJSON, *found.ResultJSON)
}

func TestTaskRepo_UpdateFailed(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	task := &model.Task{
		TaskUID:  "task_fail",
		UserID:   user.ID,
		FileID:   file.ID,
		Modality: "video",
		Status:   "processing",
	}
	err := taskRepo.Create(task)
	require.NoError(t, err)

	err = taskRepo.UpdateFailed(task.ID, "model server timeout", task.CreatedAt)
	assert.NoError(t, err)

	found, err := taskRepo.FindByTaskUID("task_fail")
	assert.NoError(t, err)
	assert.Equal(t, "failed", found.Status)
	assert.Equal(t, "model server timeout", found.ErrorMessage)
}

func TestTaskRepo_ListByUser(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	for i := 0; i < 5; i++ {
		task := &model.Task{
			TaskUID:  "task_list_" + string(rune('a'+i)),
			UserID:   user.ID,
			FileID:   file.ID,
			Modality: "image",
			Status:   "completed",
		}
		err := taskRepo.Create(task)
		require.NoError(t, err)
	}

	tasks, total, err := taskRepo.ListByUser(user.ID, 0, 3)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, tasks, 3)
}

func TestTaskRepo_SoftDelete(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	task := &model.Task{
		TaskUID:  "task_del",
		UserID:   user.ID,
		FileID:   file.ID,
		Modality: "image",
		Status:   "completed",
	}
	err := taskRepo.Create(task)
	require.NoError(t, err)

	err = taskRepo.SoftDelete("task_del", user.ID)
	assert.NoError(t, err)

	// After soft delete, should not appear in ListByUser
	tasks, total, err := taskRepo.ListByUser(user.ID, 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, tasks, 0)
}

func TestTaskRepo_BatchSoftDelete(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	uids := []string{"task_bd_1", "task_bd_2", "task_bd_3"}
	for _, uid := range uids {
		task := &model.Task{
			TaskUID:  uid,
			UserID:   user.ID,
			FileID:   file.ID,
			Modality: "audio",
			Status:   "completed",
		}
		err := taskRepo.Create(task)
		require.NoError(t, err)
	}

	affected, err := taskRepo.BatchSoftDelete(uids[:2], user.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), affected)

	tasks, total, err := taskRepo.ListByUser(user.ID, 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, tasks, 1)
}

func TestTaskRepo_FindPendingAndProcessing(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	task1 := &model.Task{TaskUID: "task_p1", UserID: user.ID, FileID: file.ID, Modality: "image", Status: "pending"}
	task2 := &model.Task{TaskUID: "task_p2", UserID: user.ID, FileID: file.ID, Modality: "image", Status: "processing"}
	task3 := &model.Task{TaskUID: "task_p3", UserID: user.ID, FileID: file.ID, Modality: "image", Status: "completed"}
	require.NoError(t, taskRepo.Create(task1))
	require.NoError(t, taskRepo.Create(task2))
	require.NoError(t, taskRepo.Create(task3))

	pending, err := taskRepo.FindPendingTasks()
	assert.NoError(t, err)
	assert.Len(t, pending, 1)

	processing, err := taskRepo.FindProcessingTasks()
	assert.NoError(t, err)
	assert.Len(t, processing, 1)
}

func TestTaskRepo_FailProcessingTasks(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	task := &model.Task{TaskUID: "task_fp", UserID: user.ID, FileID: file.ID, Modality: "image", Status: "processing"}
	require.NoError(t, taskRepo.Create(task))

	err := taskRepo.FailProcessingTasks("service restarted")
	assert.NoError(t, err)

	found, err := taskRepo.FindByTaskUID("task_fp")
	assert.NoError(t, err)
	assert.Equal(t, "failed", found.Status)
	assert.Equal(t, "service restarted", found.ErrorMessage)
}

func TestTaskRepo_StatsForUser(t *testing.T) {
	_, userRepo, fileRepo, taskRepo := setupTaskTestDB(t)
	user, file := createTestUserAndFile(t, userRepo, fileRepo)

	tasks := []*model.Task{
		{TaskUID: "task_s1", UserID: user.ID, FileID: file.ID, Modality: "image", Status: "completed"},
		{TaskUID: "task_s2", UserID: user.ID, FileID: file.ID, Modality: "image", Status: "completed"},
		{TaskUID: "task_s3", UserID: user.ID, FileID: file.ID, Modality: "video", Status: "completed"},
		{TaskUID: "task_s4", UserID: user.ID, FileID: file.ID, Modality: "audio", Status: "failed"},
	}
	for _, task := range tasks {
		require.NoError(t, taskRepo.Create(task))
	}

	total, byModality, err := taskRepo.StatsForUser(user.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, int64(2), byModality["image"])
	assert.Equal(t, int64(1), byModality["video"])
	assert.Zero(t, byModality["audio"])
}
