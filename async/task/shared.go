package task

import "github.com/TelephoneTan/GoPromise/async/promise"

type Shared[T any] struct {
	Job       promise.Job[T]
	Semaphore *promise.Semaphore
	promise   chan *promise.Type[T]
}

func (s *Shared[T]) Init() *Shared[T] {
	s.promise = make(chan *promise.Type[T], 1)
	s.promise <- nil
	return s
}

func (s *Shared[T]) Do() *promise.Type[T] {
	return (&promise.Type[T]{
		Job: promise.Job[T]{
			Do: func(rs promise.Resolver[T], re promise.Rejector) {
				p := <-s.promise
				if p == nil || p.TryAwait() {
					p = (&promise.Type[T]{
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
