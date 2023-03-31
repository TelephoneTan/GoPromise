package task

import "github.com/TelephoneTan/GoPromise/async/promise"

type Shared[T any] struct {
	Job       promise.Job[T]
	Semaphore *promise.Semaphore
	promise   chan *promise.Promise[T]
}

func (s *Shared[T]) Init() *Shared[T] {
	s.promise = make(chan *promise.Promise[T], 1)
	s.promise <- nil
	return s
}

func (s *Shared[T]) Do() *promise.Promise[T] {
	return (&promise.Promise[T]{
		Job: promise.Job[T]{
			Do: func(rs promise.Resolver[T], re promise.Rejector) {
				p := <-s.promise
				if p == nil || p.TryAwait() {
					p = (&promise.Promise[T]{
						Job:       s.Job,
						Semaphore: s.Semaphore,
					}).Init()
				}
				s.promise <- p
				rs.ResolvePromise(p)
			},
		},
	}).Init()
}
