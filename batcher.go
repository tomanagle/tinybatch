package batcher

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/tomanagle/tinybatch/internal/processor"
)

var ErrChannelClosedError = errors.New("channel closed")

type Batcher[T any, R any] struct {
	ctx context.Context
	// The maximum number of items that can be batched together

	maxBatchSize int
	// The maximum delay in milliseconds before the batch is processed
	maxBatchDelay time.Duration

	// The channel to send jobs to the processor
	jobs chan T

	// cancel internal context
	cancel context.CancelFunc

	// pendingJobs is used to wait for all the jobs to be processed
	pendingJobs sync.WaitGroup

	// processor is the internal processor
	processor interface {
		Start()
	}

	batchProcessor processor.BatchProcessor[T, R]
}

type batcherConfig struct {
	maxBatchSize  int
	maxBatchDelay time.Duration
}

type Option interface {
	applyOption(opt *batcherConfig)
}

// New creates a new Batcher with options applied.
func New[T any, R any](parentCtx context.Context, batchProcessor processor.BatchProcessor[T, R], opts ...Option) *Batcher[T, R] {
	ctx, cancel := context.WithCancel(parentCtx)

	// Create a Batcher instance with default settings
	b := &Batcher[T, R]{
		ctx:            ctx,
		maxBatchSize:   100,                      // Default maximum batch size
		maxBatchDelay:  1_000 * time.Millisecond, // Default maximum batch delay in milliseconds
		jobs:           make(chan T),
		cancel:         cancel,
		batchProcessor: batchProcessor,
	}

	// Apply provided options to configure the Batcher
	var cfg batcherConfig
	for _, option := range opts {
		option.applyOption(&cfg)
	}

	// Initialize the processor with the given context and batch processing function
	b.processor = processor.New[T, R](processor.NewProcessorParams[T, R]{
		Ctx:            b.ctx,
		MaxBatchDelay:  b.maxBatchDelay,
		Jobs:           b.jobs,
		MaxBatchSize:   b.maxBatchSize,
		BatchProcessor: b.batchProcessor,
		Wg:             &b.pendingJobs,
	})
	return b
}

type maxBatchSizeOption struct {
	maxBatchSize int
}

func (o maxBatchSizeOption) applyOption(config *batcherConfig) {
	config.maxBatchSize = o.maxBatchSize
}

// WithMaxBatchSize configures the maximum number of jobs in a batch.
func WithMaxBatchSize(maxBatchSize int) Option {
	return maxBatchSizeOption{
		maxBatchSize: maxBatchSize,
	}
}

type maxBatchDelayOption struct {
	maxBatchDelay time.Duration
}

func (o maxBatchDelayOption) applyOption(config *batcherConfig) {
	config.maxBatchDelay = o.maxBatchDelay
}

// WithMaxBatchDelay configures the maximum delay before a batch is processed.
func WithMaxBatchDelay(maxBatchDelay time.Duration) Option {
	return maxBatchDelayOption{
		maxBatchDelay: maxBatchDelay,
	}
}

// Start launches the batch processing in a goroutine.
func (b *Batcher[T, R]) Start() {
	go b.processor.Start()
}

// Add enqueues a job into the batcher.
func (b *Batcher[T, R]) Add(item T) error {
	select {
	case b.jobs <- item:
		return nil
	case <-b.ctx.Done():
		return errors.New("batcher is closed")
	}
}

// Stop halts the batcher operations and ensures the batch has finished processing all the jobs.
func (b *Batcher[T, R]) Stop() {
	b.cancel()
	b.pendingJobs.Wait()
}
