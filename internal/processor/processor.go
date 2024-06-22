package processor

import (
	"context"
	"sync"
	"time"
)

type BatchProcessor[Job any, Result any] func(jobs []Job) ([]Result, error)

type Processor[Job any, Result any] struct {
	ctx            context.Context
	maxBatchDelay  time.Duration
	jobs           <-chan Job
	maxBatchSize   int
	batchProcessor BatchProcessor[Job, Result]
	wg             *sync.WaitGroup
}

type NewProcessorParams[Job any, Result any] struct {
	Ctx            context.Context
	MaxBatchDelay  time.Duration
	Jobs           <-chan Job
	MaxBatchSize   int
	BatchProcessor BatchProcessor[Job, Result]
	Wg             *sync.WaitGroup
}

func New[Job any, Result any](params NewProcessorParams[Job, Result]) *Processor[Job, Result] {
	return &Processor[Job, Result]{
		ctx:            params.Ctx,
		maxBatchDelay:  params.MaxBatchDelay,
		jobs:           params.Jobs,
		maxBatchSize:   params.MaxBatchSize,
		batchProcessor: params.BatchProcessor,
		wg:             params.Wg,
	}
}

func (p *Processor[Job, Result]) Start() {
	timer := time.NewTimer(p.maxBatchDelay)
	defer timer.Stop()

	var batch []Job

	for {
		select {
		case <-p.ctx.Done():
			if len(batch) > 0 {
				p.processBatch(batch)
			}
			return
		case job, ok := <-p.jobs:
			p.wg.Add(1)
			if !ok {
				if len(batch) > 0 {
					p.processBatch(batch)
				}
				return
			}
			batch = append(batch, job)
			if len(batch) >= p.maxBatchSize {
				p.processBatch(batch)
				batch = nil // Reset the batch
			}
		case <-timer.C:
			if len(batch) > 0 {
				p.processBatch(batch)
				batch = nil // Reset the batch
			}
		}
	}
}

func (p *Processor[Job, Result]) processBatch(batch []Job) {
	p.batchProcessor(batch)
	for i := 0; i < len(batch); i++ {
		p.wg.Done()
	}
}
