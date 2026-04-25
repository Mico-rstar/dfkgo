package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemoryQueue_PushPop(t *testing.T) {
	q := NewMemoryQueue(10)
	ctx := context.Background()

	require.NoError(t, q.Push(ctx, "task_001"))
	require.NoError(t, q.Push(ctx, "task_002"))

	id, err := q.Pop(ctx)
	require.NoError(t, err)
	require.Equal(t, "task_001", id)

	id, err = q.Pop(ctx)
	require.NoError(t, err)
	require.Equal(t, "task_002", id)
}

func TestMemoryQueue_PushContextCancelled(t *testing.T) {
	q := NewMemoryQueue(1)
	ctx := context.Background()

	// 填满队列
	require.NoError(t, q.Push(ctx, "task_001"))

	// Push 时 context 已取消
	ctx2, cancel := context.WithCancel(context.Background())
	cancel()
	err := q.Push(ctx2, "task_002")
	require.Error(t, err)
}

func TestMemoryQueue_PopContextCancelled(t *testing.T) {
	q := NewMemoryQueue(10)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := q.Pop(ctx)
	require.Error(t, err)
}

func TestMemoryQueue_Close(t *testing.T) {
	q := NewMemoryQueue(10)
	ctx := context.Background()

	require.NoError(t, q.Push(ctx, "task_001"))
	require.NoError(t, q.Close())

	err := q.Push(ctx, "task_002")
	require.Error(t, err)
}
