package jobq

type result struct {
	res Result
	err error
}

type results struct {
	capacity int
	results  chan result
}

func newResults(capacity int) *results {
	return &results{
		capacity: capacity,
		results:  make(chan result, capacity),
	}
}

func (r *results) store(res Result, err error) {
	r.results <- result{
		res: res,
		err: err,
	}
}

func (r *results) next(res *Result, err *error) bool {
	rr, ok := <-r.results
	if !ok {
		return false
	}

	if res != nil {
		*res = rr.res
	}

	if err != nil {
		*err = rr.err
	}

	return true
}

func (r *results) close() {
	close(r.results)
}
