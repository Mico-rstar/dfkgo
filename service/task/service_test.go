package task

import (
	"context"
	"dfkgo/model"
	"dfkgo/repository"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (*TaskService, *repository.TaskRepo, *repository.FileRepo, TaskQueue) {
	db, err := repository.InitTestDB()
	require.NoError(t, err)

	taskRepo := repository.NewTaskRepo(db)
	fileRepo := repository.NewFileRepo(db)
	queue := NewMemoryQueue(100)
	svc := NewTaskService(taskRepo, fileRepo, queue)
	return svc, taskRepo, fileRepo, queue
}

func createTestFile(t *testing.T, fileRepo *repository.FileRepo, userID uint64, fileUID string) {
	file := &model.File{
		FileUID:      fileUID,
		UserID:       userID,
		FileName:     "test.jpg",
		MimeType:     "image/jpeg",
		FileSize:     1024,
		Modality:     "image",
		MD5:          "d41d8cd98f00b204e9800998ecf8427e",
		OSSBucket:    "test-bucket",
		OSSObjectKey: "files/1/test.jpg",
		OSSURL:       "https://oss.example.com/files/1/test.jpg",
		UploadStatus: "completed",
	}
	require.NoError(t, fileRepo.Create(file))
}

func TestTaskService_CreateTask(t *testing.T) {
	svc, _, fileRepo, queue := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	taskUID, err := svc.CreateTask(1, "file_abc123", "image")
	require.NoError(t, err)
	require.Contains(t, taskUID, "task_")

	// 验证入队
	id, err := queue.Pop(context.Background())
	require.NoError(t, err)
	require.Equal(t, taskUID, id)
}

func TestTaskService_CreateTask_FileNotFound(t *testing.T) {
	svc, _, _, _ := setupTestService(t)

	_, err := svc.CreateTask(1, "file_nonexistent", "image")
	require.Error(t, err)
}

func TestTaskService_CreateTask_InvalidModality(t *testing.T) {
	svc, _, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	_, err := svc.CreateTask(1, "file_abc123", "pdf")
	require.Error(t, err)
}

func TestTaskService_CreateTask_WrongUser(t *testing.T) {
	svc, _, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	_, err := svc.CreateTask(999, "file_abc123", "image")
	require.Error(t, err)
}

func TestTaskService_GetTaskStatus(t *testing.T) {
	svc, _, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	taskUID, err := svc.CreateTask(1, "file_abc123", "image")
	require.NoError(t, err)

	task, err := svc.GetTaskStatus(1, taskUID)
	require.NoError(t, err)
	require.Equal(t, "pending", task.Status)
	require.Equal(t, "image", task.Modality)
}

func TestTaskService_GetTaskStatus_NotFound(t *testing.T) {
	svc, _, _, _ := setupTestService(t)

	_, err := svc.GetTaskStatus(1, "task_nonexistent")
	require.Error(t, err)
}

func TestTaskService_CancelTask(t *testing.T) {
	svc, _, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	taskUID, err := svc.CreateTask(1, "file_abc123", "image")
	require.NoError(t, err)

	err = svc.CancelTask(1, taskUID)
	require.NoError(t, err)

	task, err := svc.GetTaskStatus(1, taskUID)
	require.NoError(t, err)
	require.Equal(t, "cancelled", task.Status)
}

func TestTaskService_CancelTask_AlreadyCompleted(t *testing.T) {
	svc, taskRepo, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	taskUID, err := svc.CreateTask(1, "file_abc123", "image")
	require.NoError(t, err)

	// 手动设置为 completed
	task, _ := taskRepo.FindByTaskUID(taskUID)
	taskRepo.UpdateStatus(task.ID, "completed")

	err = svc.CancelTask(1, taskUID)
	require.Error(t, err)
}

func TestTaskService_RecoverOrphanTasks(t *testing.T) {
	svc, taskRepo, fileRepo, queue := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	// 创建一个 processing 和一个 pending task
	taskUID1, _ := svc.CreateTask(1, "file_abc123", "image")
	// drain queue
	queue.Pop(context.Background())

	task1, _ := taskRepo.FindByTaskUID(taskUID1)
	taskRepo.UpdateStatus(task1.ID, "processing")

	// 创建第二个文件和 task（pending）
	file2 := &model.File{
		FileUID: "file_def456", UserID: 1, FileName: "test2.jpg",
		MimeType: "image/jpeg", FileSize: 2048, Modality: "image",
		MD5: "a41d8cd98f00b204e9800998ecf8427e", OSSBucket: "test-bucket",
		OSSObjectKey: "files/1/test2.jpg", OSSURL: "https://oss.example.com/files/1/test2.jpg",
		UploadStatus: "completed",
	}
	fileRepo.Create(file2)
	taskUID2, _ := svc.CreateTask(1, "file_def456", "image")
	// drain queue
	queue.Pop(context.Background())

	err := svc.RecoverOrphanTasks()
	require.NoError(t, err)

	// processing 应该变成 failed
	t1, _ := taskRepo.FindByTaskUID(taskUID1)
	require.Equal(t, "failed", t1.Status)
	require.Equal(t, "service restarted", t1.ErrorMessage)

	// pending 应该重新入队
	id, err := queue.Pop(context.Background())
	require.NoError(t, err)
	require.Equal(t, taskUID2, id)
}

func TestTaskService_ListHistory(t *testing.T) {
	svc, _, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	svc.CreateTask(1, "file_abc123", "image")

	items, total, err := svc.ListHistoryWithFiles(1, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
}

func TestTaskService_DeleteHistory(t *testing.T) {
	svc, _, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	taskUID, _ := svc.CreateTask(1, "file_abc123", "image")

	err := svc.DeleteHistory(1, taskUID)
	require.NoError(t, err)

	// 软删后不应出现在列表中
	items, total, err := svc.ListHistoryWithFiles(1, 1, 10)
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.Len(t, items, 0)
}

func TestTaskService_BatchDeleteHistory(t *testing.T) {
	svc, _, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	file2 := &model.File{
		FileUID: "file_def456", UserID: 1, FileName: "test2.jpg",
		MimeType: "image/jpeg", FileSize: 2048, Modality: "image",
		MD5: "b41d8cd98f00b204e9800998ecf8427e", OSSBucket: "test-bucket",
		OSSObjectKey: "files/1/test2.jpg", OSSURL: "https://oss.example.com/files/1/test2.jpg",
		UploadStatus: "completed",
	}
	fileRepo.Create(file2)

	taskUID1, _ := svc.CreateTask(1, "file_abc123", "image")
	taskUID2, _ := svc.CreateTask(1, "file_def456", "image")

	count, err := svc.BatchDeleteHistory(1, []string{taskUID1, taskUID2})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
}

func TestTaskService_GetStats(t *testing.T) {
	svc, _, fileRepo, _ := setupTestService(t)
	createTestFile(t, fileRepo, 1, "file_abc123")

	svc.CreateTask(1, "file_abc123", "image")

	total, byModality, byCategory, err := svc.GetStats(1)
	require.NoError(t, err)
	// no completed tasks yet
	require.Equal(t, int64(0), total)
	require.NotNil(t, byModality)
	require.NotNil(t, byCategory)
}

// MockModelClient for worker tests
type MockModelClient struct {
	result []byte
	err    error
}

func (m *MockModelClient) Detect(ctx context.Context, modality Modality, ossURL string, taskID string, userID string) ([]byte, error) {
	return m.result, m.err
}
