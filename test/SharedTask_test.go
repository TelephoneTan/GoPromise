package test

import (
	"github.com/TelephoneTan/GoPromise/async/promise"
	"github.com/TelephoneTan/GoPromise/async/task"
	"github.com/TelephoneTan/GoPromise/util"
	"testing"
	"time"
)

func TestSharedTask(t *testing.T) {
	task0 := (&task.Shared[string]{
		Job: promise.Job[string]{
			Do: func(rs promise.Resolver[string], re promise.Rejector) {
				time.Sleep(1 * time.Second)
				rs.ResolveValue(util.Ptr(time.Now().String()))
			},
		},
	}).Init()
	var all []*promise.Type[any]
	(&task.Timed{
		Job: promise.Job[bool]{
			Do: func(rs promise.Resolver[bool], re promise.Rejector) {
				all = append(all, promise.Then(task0.Do(), promise.FulfilledListener[string, any]{
					OnFulfilled: func(v *string) any {
						println(*v)
						return nil
					},
				}))
				rs.ResolveValue(util.Ptr(true))
			},
		},
	}).Init(200*time.Millisecond, 50).Start().Await()
	promise.AwaitAll(all)
}
