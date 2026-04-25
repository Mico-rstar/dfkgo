package task

import (
	"context"
	"dfkgo/repository"
	"fmt"
	"log"
	"sync"
	"time"
)

type WorkerPool struct {
	queue      TaskQueue
	taskRepo   *repository.TaskRepo
	fileRepo   *repository.FileRepo
	client     ModelClient
	poolSize   int
	wg         sync.WaitGroup
	cancelFunc context.CancelFunc
}

func NewWorkerPool(queue TaskQueue, taskRepo *repository.TaskRepo, fileRepo *repository.FileRepo, client ModelClient, poolSize int) *WorkerPool {
	return &WorkerPool{
		queue:    queue,
		taskRepo: taskRepo,
		fileRepo: fileRepo,
		client:   client,
		poolSize: poolSize,
	}
}

func (wp *WorkerPool) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	wp.cancelFunc = cancel
	for i := 0; i < wp.poolSize; i++ {
		wp.wg.Add(1)
		go wp.run(ctx, i)
	}
	log.Printf("[WorkerPool] started %d workers", wp.poolSize)
}

func (wp *WorkerPool) Stop() {
	wp.cancelFunc()
	wp.queue.Close()
	wp.wg.Wait()
	log.Println("[WorkerPool] all workers stopped")
}

func (wp *WorkerPool) run(ctx context.Context, workerID int) {
	defer wp.wg.Done()
	for {
		taskUID, err := wp.queue.Pop(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[Worker-%d] pop error: %v", workerID, err)
			return
		}
		wp.processTask(ctx, workerID, taskUID)
	}
}

func (wp *WorkerPool) processTask(ctx context.Context, workerID int, taskUID string) {
	// 1. 查 DB 确认 status 仍为 pending
	task, err := wp.taskRepo.FindByTaskUID(taskUID)
	if err != nil {
		log.Printf("[Worker-%d] task %s not found: %v", workerID, taskUID, err)
		return
	}
	if task.Status != "pending" {
		log.Printf("[Worker-%d] task %s status is %s, skip", workerID, taskUID, task.Status)
		return
	}

	// 2. 置 processing
	now := time.Now()
	if err := wp.taskRepo.SetProcessing(task.ID, now); err != nil {
		log.Printf("[Worker-%d] set processing failed: %v", workerID, err)
		return
	}

	// 3. 查文件获取 OSS URL
	file, err := wp.fileRepo.FindByID(task.FileID)
	if err != nil {
		wp.taskRepo.UpdateFailed(task.ID, fmt.Sprintf("file not found: %v", err), time.Now())
		return
	}

	// 4. 调 ModelServer
	log.Printf("[Worker-%d] detecting task %s, modality=%s", workerID, taskUID, task.Modality)
	resultJSON, err := wp.client.Detect(ctx, Modality(task.Modality), file.OSSURL, task.TaskUID, fmt.Sprintf("%d", task.UserID))
	if err != nil {
		wp.taskRepo.UpdateFailed(task.ID, err.Error(), time.Now())
		log.Printf("[Worker-%d] task %s detect failed: %v", workerID, taskUID, err)
		return
	}

	// 5. 检查是否被取消（软取消语义：processing 态被取消后丢弃结果）
	task, err = wp.taskRepo.FindByTaskUID(taskUID)
	if err == nil && task.Status == "cancelled" {
		log.Printf("[Worker-%d] task %s was cancelled, discarding result", workerID, taskUID)
		return
	}

	// 6. 写结果
	result := string(resultJSON)
	if err := wp.taskRepo.UpdateResult(task.ID, "completed", result, time.Now()); err != nil {
		log.Printf("[Worker-%d] update result failed: %v", workerID, err)
	} else {
		log.Printf("[Worker-%d] task %s completed", workerID, taskUID)
	}
}
