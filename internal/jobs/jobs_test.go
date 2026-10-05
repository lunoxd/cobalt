package jobs

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPool_EnqueueAndProcess(t *testing.T) {
	pool := NewPool(2, 10)
	pool.Start()
	defer pool.Stop()

	var wg sync.WaitGroup
	wg.Add(2)

	var processedTypes []string
	var mu sync.Mutex

	pool.Register("test_task", func(ctx context.Context, job *Job) error {
		mu.Lock()
		processedTypes = append(processedTypes, job.Type)
		mu.Unlock()
		wg.Done()
		return nil
	})

	_, err := pool.Enqueue("test_task", map[string]any{"data": "one"})
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	_, err = pool.Enqueue("test_task", map[string]any{"data": "two"})
	if err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		mu.Lock()
		count := len(processedTypes)
		mu.Unlock()
		if count != 2 {
			t.Errorf("expected 2 processed jobs, got %d", count)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout waiting for background jobs to process")
	}
}
