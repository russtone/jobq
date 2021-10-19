package jobq_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/russtone/jobq"
)

var errTest = errors.New("test")

type DoMock struct {
	mock.Mock
	progress   int
	progressWG sync.WaitGroup
	wait       sync.WaitGroup
}

//
// Task
//

type Task struct {
	ID    int
	Retry bool
	Err   bool
}

func (t *Task) Task() {}

var _ jobq.Task = new(Task)

//
// Result
//

type Result struct {
	task *Task
}

func (r *Result) Result() {}

var _ jobq.Result = new(Result)

//
// DoFunc
//

func (do *DoMock) Do(task jobq.Task) (jobq.Result, error) {

	do.Called(task)

	t, ok := task.(*Task)
	if !ok {
		return nil, errors.New("cast failed")
	}

	defer func() {
		if t.ID < do.progress {
			do.progressWG.Done()
		}
	}()

	if t.ID >= do.progress {
		do.wait.Wait()
	}

	// Error.
	if t.Err {
		t.Err = false
		return nil, errTest
	}

	if t.Retry {
		t.Retry = false
		return nil, jobq.Retry(nil)
	}

	return &Result{task: t}, nil
}

func TestJobqueue(t *testing.T) {

	tests := []struct {
		jobs     int
		workers  int
		errID    int
		retryID  int
		progress int
	}{
		{jobs: 10, workers: 3, errID: 5, retryID: 6, progress: 5},
		{jobs: 100, workers: 5, errID: 5, retryID: 50, progress: 10},
		{jobs: 100, workers: 50, errID: 50, retryID: 80, progress: 80},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {

			if tt.progress > tt.jobs {
				panic("invald test case: progress must be less then jobs count")
			}

			if tt.errID > tt.jobs || tt.retryID > tt.jobs {
				panic("invalid test case: id must be less then jobs count")
			}

			do := &DoMock{
				progress: tt.progress,
			}

			do.progressWG.Add(tt.progress)

			for i := 0; i < tt.jobs; i++ {
				if i == tt.retryID {
					do.On("Do", &Task{ID: i, Retry: true}).Return().Once()
				} else if i == tt.errID {
					do.On("Do", &Task{ID: i, Err: true}).Return().Once()
					continue
				}
				do.On("Do", &Task{ID: i}).Return().Once()
			}

			queue := jobq.New(do.Do, tt.workers, tt.jobs)

			queue.Start()

			wg := sync.WaitGroup{}
			wg.Add(1)

			// Process results
			go func() {
				defer wg.Done()

				var (
					res jobq.Result
					err error
				)

				done := 0
				errs := 0

				for queue.Next(&res, &err) {
					if err != nil {
						// The only error that could happen is testErr.
						assert.Error(t, errTest, err)
						errs++
						continue
					}

					done++
				}

				// All jobs must be proccessed.
				assert.Equal(t, tt.jobs-1, done)

				// There must be only 1 error.
				assert.Equal(t, 1, errs)
			}()

			do.wait.Add(1)

			for i := 0; i < tt.jobs; i++ {
				task := &Task{
					ID:    i,
					Retry: i == tt.retryID,
					Err:   i == tt.errID,
				}

				queue.Add(task)
			}

			// Wait for jobs to finish to compare progress.
			do.progressWG.Wait()
			assert.Equal(t, float64(tt.progress)/float64(tt.jobs), queue.Progress())

			// Unlock rest of jobs.
			do.wait.Done()

			queue.Wait()

			queue.Close()

			// Wait erors and output processors.
			wg.Wait()

			do.AssertExpectations(t)
		})
	}

}
