package jobq

import (
	"sync"
	"sync/atomic"
	"time"
)

type counter struct {
	wg sync.WaitGroup

	count     uint64
	processed uint64

	createdAt time.Time
}

func newCounter() *counter {
	return &counter{
		createdAt: time.Now(),
	}
}

func (j *counter) progress() float64 {
	processed := atomic.LoadUint64(&j.processed)
	count := atomic.LoadUint64(&j.count)
	return float64(processed) / float64(count)
}

func (j *counter) speed() float64 {
	processed := atomic.LoadUint64(&j.processed)
	return float64(processed) / time.Since(j.createdAt).Seconds()
}

func (j *counter) wait() {
	j.wg.Wait()
}

func (j *counter) incCount() {
	atomic.AddUint64(&j.count, 1)
	j.wg.Add(1)
}

func (j *counter) incProcessed() {
	atomic.AddUint64(&j.processed, 1)
	j.wg.Done()
}
