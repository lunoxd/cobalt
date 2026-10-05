package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Job represents a background task.
type Job struct {
	ID        uuid.UUID      `json:"id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
	Attempts  int            `json:"attempts"`
	MaxRetry  int            `json:"max_retry"`
}

// HandlerFunc is the execution signature for a job type.
type HandlerFunc func(ctx context.Context, job *Job) error

// Pool implements an in-process, non-blocking background worker pool.
type Pool struct {
	queue       chan *Job
	handlers    map[string]HandlerFunc
	handlersMu  sync.RWMutex
	workerCount int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewPool initializes a background worker pool with a buffer and worker count.
func NewPool(workerCount, queueSize int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		queue:       make(chan *Job, queueSize),
		handlers:    make(map[string]HandlerFunc),
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Register registers a handler for a job type.
func (p *Pool) Register(jobType string, handler HandlerFunc) {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	p.handlers[jobType] = handler
}

// Start spawns the background worker goroutines.
func (p *Pool) Start() {
	slog.Info("Starting background worker pool", "workers", p.workerCount)
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop gracefully signals workers to stop and waits for pending jobs to finish.
func (p *Pool) Stop() {
	slog.Info("Stopping background worker pool...")
	p.cancel()
	close(p.queue)
	p.wg.Wait()
	slog.Info("Background worker pool stopped")
}

// Enqueue submits a job to the queue. Returns an error if queue is full.
func (p *Pool) Enqueue(jobType string, payload map[string]any) (*Job, error) {
	job := &Job{
		ID:        uuid.New(),
		Type:      jobType,
		Payload:   payload,
		CreatedAt: time.Now(),
		MaxRetry:  3,
	}

	select {
	case p.queue <- job:
		return job, nil
	default:
		return nil, fmt.Errorf("job queue is full, could not enqueue job %s", jobType)
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			// Drain remaining jobs before exiting
			for job := range p.queue {
				p.processJob(job)
			}
			return
		case job, ok := <-p.queue:
			if !ok {
				return
			}
			p.processJob(job)
		}
	}
}

func (p *Pool) processJob(job *Job) {
	p.handlersMu.RLock()
	handler, exists := p.handlers[job.Type]
	p.handlersMu.RUnlock()

	if !exists {
		slog.Warn("No handler registered for job type", "type", job.Type, "job_id", job.ID)
		return
	}

	job.Attempts++
	slog.Info("Processing background job", "type", job.Type, "job_id", job.ID, "attempt", job.Attempts)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := handler(ctx, job); err != nil {
		slog.Error("Job failed", "type", job.Type, "job_id", job.ID, "err", err, "attempt", job.Attempts)
		// Retry logic
		if job.Attempts < job.MaxRetry {
			go func() {
				time.Sleep(time.Duration(job.Attempts*2) * time.Second)
				select {
				case p.queue <- job:
				default:
					slog.Error("Failed to re-enqueue failed job (queue full)", "job_id", job.ID)
				}
			}()
		}
		return
	}

	slog.Info("Job completed successfully", "type", job.Type, "job_id", job.ID)
}

// QueueStats returns current queue load metrics.
func (p *Pool) QueueStats() map[string]any {
	return map[string]any{
		"queued_jobs":  len(p.queue),
		"queue_cap":    cap(p.queue),
		"worker_count": p.workerCount,
	}
}
