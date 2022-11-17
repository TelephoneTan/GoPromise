package task

import (
	"github.com/TelephoneTan/GoPromise/async/promise"
	"sync"
)

type Once[T any] struct {
	Job       promise.Job[T]
	Semaphore *promise.Semaphore
	once      sync.Once
	promise   *promise.Type[T]
}

func (o *Once[T]) Init() *Once[T] {
	return o
}

func (o *Once[T]) Do() *promise.Type[T] {
	return (&promise.Type[T]{
		Job: promise.Job[T]{
			Do: func(rs promise.Resolver[T], re promise.Rejector) {
				o.once.Do(func() {
					o.promise = (&promise.Type[T]{
						Job:       o.Job,
						Semaphore: o.Semaphore,
					}).Init()
				})
				rs.ResolvePromise(o.promise)
			},
		},
	}).Init()
}
