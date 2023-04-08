package task

import "github.com/TelephoneTan/GoPromise/async/promise"

type _Shared[T any] struct {
	Job       promise.Job[T]
	Semaphore promise.Semaphore
	promise   chan promise.Promise[T]
}

type Shared[T any] struct {
	*_Shared[T]
}

func NewSharedTask[T any](job promise.Job[T]) Shared[T] {
	return Shared[T]{&_Shared[T]{Job: job}}.init()
}

func NewSharedTaskWithSemaphore[T any](job promise.Job[T], semaphore promise.Semaphore) Shared[T] {
	return Shared[T]{&_Shared[T]{Job: job, Semaphore: semaphore}}.init()
}

func (s Shared[T]) init() Shared[T] {
	s.promise = make(chan promise.Promise[T], 1)
	s.promise <- promise.Promise[T]{}
	return s
}

func (s Shared[T]) Do() promise.Promise[T] {
	return promise.NewPromise(promise.Job[T]{
		Do: func(rs promise.Resolver[T], re promise.Rejector) {
			p := <-s.promise
			if p == (promise.Promise[T]{}) || p.TryAwait() {
				p = promise.NewPromiseWithSemaphore(s.Job, s.Semaphore)
			}
			s.promise <- p
			rs.ResolvePromise(p)
		},
	})
}
