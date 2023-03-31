package test

import (
	"fmt"
	"github.com/TelephoneTan/GoPromise/async/promise"
	"github.com/TelephoneTan/GoPromise/util"
	"testing"
	"time"
)

var (
	userB,
	userA,
	hostB,
	hostA,
	user,
	host,
	total promise.Semaphore
)

func init() {
	total.Init(10).
		Then(
			host.Init(5).
				Then(
					hostA.Init(3),
				).
				Then(
					hostB.Init(3),
				),
		).
		Then(
			user.Init(5).
				Then(
					userA.Init(3),
				).
				Then(
					userB.Init(3),
				),
		)
}

func launch[T any](all []*promise.Promise[T], n int, semaphore *promise.Semaphore, name string) []*promise.Promise[T] {
	for i := 0; i < n; i++ {
		ii := i
		makeStr := (&promise.Promise[string]{
			Semaphore: semaphore,
			Job: promise.Job[string]{
				Do: func(resolver promise.Resolver[string], rejector promise.Rejector) {
					time.Sleep(5 * time.Second)
					resolver.ResolveValue(
						util.Ptr(
							fmt.Sprintf("%s %d -> %s", name, ii, time.Now().String()),
						),
					)
				},
			},
		}).Init()
		printStr := promise.Then(makeStr, promise.FulfilledListener[string, T]{
			OnFulfilled: func(value *string) any {
				println(*value)
				return nil
			},
		})
		all = append(all, printStr)
	}
	return all
}

func TestMultipleSemaphore(t *testing.T) {
	var all []*promise.Promise[string]
	all = launch(all, 10, &hostA, "hostA")
	all = launch(all, 10, &hostB, "hostB")
	all = launch(all, 10, &userA, "userA")
	all = launch(all, 10, &userB, "userB")
	promise.FinallyRequired[any](promise.SettledListener[any]{
		OnSettled: func() *promise.Promise[any] {
			println("结束了")
			return nil
		},
	}, all).Await()
}
