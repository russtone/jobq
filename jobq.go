// Package jobq
package jobq

import (
	"sync"
)

// DoFunc is a type for function which is processing tasks.
type DoFunc func(task Task) (Result, error)

// Task represents task to add to queue.
type Task interface {
	// Task is used only for type safety.
	Task()
}

// Result represents result of the task.
type Result interface {
	// Result is used only for type safety.
	Result()
}

// TaskGroup represents group of tasks to process.
type TaskGroup interface {
	// Add adds task to group.
	Add(Task)

	// Next returns next task Result or error.
	Next(*Result, *error) bool

	// Wait blocks until all tasks in group are processed.
	Wait()

	// Close closes all internal resources.
	// Blocks until all internal goroutines are stopped.
	Close()

	// Speed returns processing speed (tasks per second).
	Speed() float64

	// Progress returns progress as number from 0 to 1.
	Progress() float64
}

// Queue represents tasks queue.
type Queue interface {
	TaskGroup

	// Start starts task processing workers.
	Start()

	// Group returns new TaskGroup to use.
	Group() TaskGroup
}

// queue is a task queue.
type queue struct {
	// do is a func which takes Task as input and
	// returns Result or error.
	do DoFunc

	// capacity is a size of TODO buffer.
	capacity int

	// workersCount is a count of workers to process tasks.
	workersCount int

	// workersWG is used to wait for workers to finish.
	workersWG sync.WaitGroup

	// todo is a channel of ready to process jobs.
	todo chan *job

	// toretry is a channel of jobs to retry.
	toretry chan *job

	// results is a storage for task results.
	results *results

	// counter is a helper type to stats (speed, progress, etc.).
	counter *counter
}

var _ Queue = new(queue)

// New is a constructor for Queue.
func New(do DoFunc, workers int, capacity int) Queue {
	return &queue{
		do:           do,
		capacity:     capacity,
		workersCount: workers,
		todo:         make(chan *job, capacity),
		toretry:      make(chan *job, capacity),
		results:      newResults(capacity),
		counter:      newCounter(),
	}
}

// Start allows queue to implement Queue interface.
func (q *queue) Start() {
	for i := 0; i < q.workersCount; i++ {
		q.workersWG.Add(1)
		go q.worker(i)
	}
}

// Close allows queue to implement Queue interface.
func (q *queue) Close() {
	close(q.todo)
	q.workersWG.Wait()
	close(q.toretry)
	q.results.close()
}

// Wait allows queue to implement Queue interface.
func (q *queue) Wait() {
	q.counter.wait()
}

// Add allows queue to implement Queue interface.
func (q *queue) Add(task Task) {
	q.add(q.job(task))
}

// Next allows queue to implement Queue interface.
func (q *queue) Next(res *Result, err *error) bool {
	return q.results.next(res, err)
}

// Progress allows queue to implement Queue interface.
func (q *queue) Progress() float64 {
	return q.counter.progress()
}

// Speed allows queue to implement Queue interface.
func (q *queue) Speed() float64 {
	return q.counter.speed()
}

// Group allows queue to implement Queue interface.
func (q *queue) Group() TaskGroup {
	return &group{
		queue:   q,
		results: newResults(q.capacity),
		counter: newCounter(),
	}
}

// job wraps Task into job.
func (q *queue) job(task Task) *job {
	return newJob(task, q.counter, q.results, nil)
}

// add adds job to queue.
func (q *queue) add(j *job) {
	j.onAdd()
	q.todo <- j
}

// worker is a task processing worker.
func (q *queue) worker(id int) {
	defer q.workersWG.Done()

loop:
	for {
		select {
		case j := <-q.toretry:
			if j != nil {
				q.process(j)
			}
		default:
			select {
			case j := <-q.todo:
				if j != nil {
					q.process(j)
				} else {
					break loop
				}
			default:
				break
			}
		}
	}
}

// process processes task using DoFunc and handles result.
func (q *queue) process(j *job) {
	res, err := q.do(j.task)

	if err != nil {
		if _, ok := err.(*retryable); ok {
			q.retry(j)
			return
		}
	}

	j.onResult(res, err)
}

// retry adds job back into todo queue.
func (q *queue) retry(j *job) {
	q.toretry <- j
}
