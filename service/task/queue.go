package task

import (
	"context"
	"errors"
	"sync"
)

var ErrQueueClosed = errors.New("queue is closed")

type TaskQueue interface {
	Push(ctx context.Context, taskID string) error
	Pop(ctx context.Context) (taskID string, err error)
	Close() error
}

type MemoryQueue struct {
	ch       chan string
	closed   chan struct{}
	closeOnce sync.Once
}

func NewMemoryQueue(capacity int) *MemoryQueue {
	return &MemoryQueue{
		ch:     make(chan string, capacity),
		closed: make(chan struct{}),
	}
}

func (q *MemoryQueue) Push(ctx context.Context, taskID string) error {
	select {
	case <-q.closed:
		return ErrQueueClosed
	default:
	}
	select {
	case q.ch <- taskID:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-q.closed:
		return ErrQueueClosed
	}
}

func (q *MemoryQueue) Pop(ctx context.Context) (string, error) {
	select {
	case id := <-q.ch:
		return id, nil
	case <-ctx.Done():
		return "", ctx.Err()
	case <-q.closed:
		return "", ErrQueueClosed
	}
}

func (q *MemoryQueue) Close() error {
	q.closeOnce.Do(func() {
		close(q.closed)
	})
	return nil
}
