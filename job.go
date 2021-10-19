package jobq

type job struct {
	task Task

	counter *counter
	results *results

	wrapped *job
}

func newJob(task Task, jc *counter, rs *results, j *job) *job {
	return &job{
		task:    task,
		counter: jc,
		results: rs,
		wrapped: j,
	}
}

func (j *job) onAdd() {
	j.counter.incCount()

	if j.wrapped != nil {
		j.wrapped.onAdd()
	}
}

func (j *job) onResult(res Result, err error) {
	j.counter.incProcessed()

	j.results.store(res, err)

	if j.wrapped != nil {
		j.wrapped.onResult(res, err)
	}
}
