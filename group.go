package jobq

var _ TaskGroup = new(group)

type group struct {
	queue   *queue
	counter *counter
	results *results
}

func (g *group) Add(task Task) {
	job := g.queue.job(task)
	g.queue.add(newJob(task, g.counter, g.results, job))
}

func (g *group) Next(res *Result, err *error) bool {
	return g.results.next(res, err)
}

func (g *group) Progress() float64 {
	return g.counter.progress()
}

func (g *group) Speed() float64 {
	return g.counter.speed()
}

func (g *group) Wait() {
	g.counter.wait()
}

func (g *group) Close() {
	g.results.close()
}
