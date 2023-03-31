package promise

import (
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type Promise[T any] struct {
	value     *T
	succeeded bool
	reason    error
	failed    bool
	cancelled bool
	timeoutSN atomic.Uint64
	settled   line
	settleIt  sync.Once
	Semaphore *Semaphore
	Job       Job[T]
}

func (t *Promise[T]) init(wrapJobWithSemaphore bool, start bool) *Promise[T] {
	if start {
		defer t.start(wrapJobWithSemaphore)
	}
	t.settled = make(line)
	return t
}

func (t *Promise[T]) Init() *Promise[T] {
	return t.init(true, true)
}

func (t *Promise[T]) settle(assign func()) (ok bool) {
	t.settleIt.Do(func() {
		ok = true
		assign()
		if t.Semaphore != nil {
			t.Semaphore.Release()
		}
		close(t.settled)
	})
	return ok
}

func (t *Promise[T]) succeed(value *T) *Promise[T] {
	t.settle(func() {
		t.value = value
		t.succeeded = true
	})
	return t
}

func (t *Promise[T]) fail(reason error) *Promise[T] {
	t.settle(func() {
		t.reason = reason
		t.failed = true
	})
	return t
}

func (t *Promise[T]) Cancel() bool {
	return t.settle(func() {
		t.cancelled = true
	})
}

func (t *Promise[T]) Await() *Promise[T] {
	<-t.settled
	return t
}

func AwaitAll[T any](all []*Promise[T]) {
	for _, p := range all {
		p.Await()
	}
}

func (t *Promise[T]) TryAwait() bool {
	select {
	case <-t.settled:
		return true
	default:
		return false
	}
}

func (t *Promise[T]) copyStateTo(tt *Promise[T]) {
	t.Await()
	if t.cancelled {
		tt.Cancel()
	}
	if t.succeeded {
		tt.succeed(t.value)
	}
	if t.failed {
		tt.fail(t.reason)
	}
}

func (t *Promise[T]) Resolve(valueOrPromise any) {
	switch x := valueOrPromise.(type) {
	case nil:
		t.ResolveValue(nil)
	case *T:
		t.ResolveValue(x)
	case *Promise[T]:
		t.ResolvePromise(x)
	}
}

func (t *Promise[T]) ResolveValue(value *T) {
	t.succeed(value)
}

func (t *Promise[T]) ResolvePromise(promise *Promise[T]) {
	go func() {
		promise.copyStateTo(t)
	}()
}

func (t *Promise[T]) cancel() {
	t.Cancel()
}

func (t *Promise[T]) Reject(e error) {
	t.fail(e)
}

func (t *Promise[T]) start(wrapJobWithSemaphore bool) {
	if t.Job.Do != nil {
		go func() {
			debug.SetPanicOnFault(true)
			ok := false
			defer func() {
				if !ok {
					a := recover()
					if a == nil {
						t.fail(nil)
					} else if e, ok := a.(error); ok {
						t.fail(e)
					} else {
						t.fail(fmt.Errorf("%#v", a))
					}
				}
			}()
			if wrapJobWithSemaphore && t.Semaphore != nil {
				t.Semaphore.Acquire()
			}
			t.Job.Do(t, t)
			ok = true
		}()
	}
}

func (t *Promise[T]) SetTimeout(d time.Duration, onTimeOut ...*TimeOutListener) *Promise[T] {
	if t.TryAwait() {
		return t
	}
	go func(sn uint64) {
		time.Sleep(d)
		if t.timeoutSN.Load() == sn && t.Cancel() && len(onTimeOut) > 0 && onTimeOut[0].OnTimeOut != nil {
			onTimeOut[0].OnTimeOut(d)
		}
	}(t.timeoutSN.Add(1))
	return t
}

func settleAll[S any](promiseList []*Promise[S], cancelledFlag []bool, succeededFlag []bool, value []*S, reason []error) {
	for i, promise := range promiseList {
		promise.Await()
		if promise.cancelled {
			cancelledFlag[i] = true
		}
		if promise.succeeded {
			value[i] = promise.value
			succeededFlag[i] = true
		}
		if promise.failed {
			reason[i] = promise.reason
			succeededFlag[i] = false
		}
	}
}

func dependOn[REQUIRED any, OPTIONAL any, SUPPLY any, T any](
	semaphore *Semaphore,
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
	f *CompoundFulfilledListener[REQUIRED, OPTIONAL, SUPPLY],
	r *RejectedListener[SUPPLY],
	s *SettledListener[T],
	c *CancelledListener,
) *Promise[SUPPLY] {
	return (&Promise[SUPPLY]{
		Semaphore: semaphore,
		Job: Job[SUPPLY]{
			Do: func(resolver Resolver[SUPPLY], rejector Rejector) {
				requiredNum := len(requiredPromise)
				requiredValue := make([]*REQUIRED, requiredNum)
				requiredReason := make([]error, requiredNum)
				requiredCancelledFlag := make([]bool, requiredNum)
				requiredSucceededFlag := make([]bool, requiredNum)
				settleAll(requiredPromise, requiredCancelledFlag, requiredSucceededFlag, requiredValue, requiredReason)
				//
				optionalNum := len(optionalPromise)
				optionalValue := make([]*OPTIONAL, optionalNum)
				optionalReason := make([]error, optionalNum)
				optionalCancelledFlag := make([]bool, optionalNum)
				optionalSucceededFlag := make([]bool, optionalNum)
				settleAll(optionalPromise, optionalCancelledFlag, optionalSucceededFlag, optionalValue, optionalReason)
				//
				succeeded := true
				cancelled := false
				var reason error
				for _, cf := range requiredCancelledFlag {
					if cf {
						cancelled = true
						break
					}
				}
				if !cancelled {
					for i, sf := range requiredSucceededFlag {
						if !sf {
							reason = requiredReason[i]
							succeeded = false
							break
						}
					}
				}
				//
				if semaphore != nil {
					semaphore.Acquire()
				}
				if cancelled {
					rejector.cancel()
					if c != nil && c.OnCancelled != nil {
						go c.OnCancelled()
					}
				}
				if s != nil && s.OnSettled != nil {
					promise := s.OnSettled()
					if promise != nil {
						promise.Await()
						if promise.cancelled {
							rejector.cancel()
						}
						if promise.failed {
							rejector.Reject(promise.reason)
						}
					}
				}
				if !cancelled {
					if succeeded {
						if f == nil || f.OnFulfilled == nil {
							resolver.ResolveValue(nil)
						} else {
							res := f.OnFulfilled(&CompoundResult[REQUIRED, OPTIONAL]{
								RequiredValue:         requiredValue,
								OptionalValue:         optionalValue,
								OptionalReason:        optionalReason,
								OptionalCancelledFlag: optionalCancelledFlag,
								OptionalSucceededFlag: optionalSucceededFlag,
							})
							if res == nil {
								resolver.ResolveValue(nil)
							} else if resP, ok := res.(*Promise[SUPPLY]); ok {
								resolver.ResolvePromise(resP)
							} else {
								resolver.ResolveValue(res.(*SUPPLY))
							}
						}
					} else {
						if r == nil || r.OnRejected == nil {
							panic(reason)
						} else {
							res := r.OnRejected(reason)
							if res == nil {
								resolver.ResolveValue(nil)
							} else if resP, ok := res.(*Promise[SUPPLY]); ok {
								resolver.ResolvePromise(resP)
							} else {
								resolver.ResolveValue(res.(*SUPPLY))
							}
						}
					}
				}
			},
		},
	}).init(false, true)
}

func ThenAllSemaphore[SUPPLY any, REQUIRED any, OPTIONAL any](
	semaphore *Semaphore,
	onFulfilled CompoundFulfilledListener[REQUIRED, OPTIONAL, SUPPLY],
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
) *Promise[SUPPLY] {
	return dependOn[REQUIRED, OPTIONAL, SUPPLY, any](
		semaphore,
		requiredPromise,
		optionalPromise,
		&onFulfilled,
		nil,
		nil,
		nil,
	)
}

func ThenAll[SUPPLY any, REQUIRED any, OPTIONAL any](
	onFulfilled CompoundFulfilledListener[REQUIRED, OPTIONAL, SUPPLY],
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
) *Promise[SUPPLY] {
	return ThenAllSemaphore(nil, onFulfilled, requiredPromise, optionalPromise)
}

func ThenRequiredSemaphore[SUPPLY any, NEED any](
	semaphore *Semaphore,
	onFulfilled CompoundFulfilledListener[NEED, any, SUPPLY],
	requiredPromise []*Promise[NEED],
) *Promise[SUPPLY] {
	return ThenAllSemaphore(semaphore, onFulfilled, requiredPromise, nil)
}

func ThenRequired[SUPPLY any, NEED any](
	onFulfilled CompoundFulfilledListener[NEED, any, SUPPLY],
	requiredPromise []*Promise[NEED],
) *Promise[SUPPLY] {
	return ThenRequiredSemaphore(nil, onFulfilled, requiredPromise)
}

func ThenSemaphore[SUPPLY any, NEED any](
	promise *Promise[NEED],
	semaphore *Semaphore,
	onFulfilled FulfilledListener[NEED, SUPPLY],
) *Promise[SUPPLY] {
	return ThenRequiredSemaphore(semaphore, CompoundFulfilledListener[NEED, any, SUPPLY]{
		OnFulfilled: func(cv *CompoundResult[NEED, any]) (res any) {
			if onFulfilled.OnFulfilled != nil {
				res = onFulfilled.OnFulfilled(cv.RequiredValue[0])
			}
			return res
		},
	}, []*Promise[NEED]{promise})
}

func Then[SUPPLY any, NEED any](
	promise *Promise[NEED],
	onFulfilled FulfilledListener[NEED, SUPPLY],
) *Promise[SUPPLY] {
	return ThenSemaphore(promise, nil, onFulfilled)
}

func CatchAllSemaphore[SUPPLY any, REQUIRED any, OPTIONAL any](
	semaphore *Semaphore,
	onRejected RejectedListener[SUPPLY],
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
) *Promise[SUPPLY] {
	return dependOn[REQUIRED, OPTIONAL, SUPPLY, any](
		semaphore,
		requiredPromise,
		optionalPromise,
		nil,
		&onRejected,
		nil,
		nil,
	)
}

func CatchAll[SUPPLY any, REQUIRED any, OPTIONAL any](
	onRejected RejectedListener[SUPPLY],
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
) *Promise[SUPPLY] {
	return CatchAllSemaphore(nil, onRejected, requiredPromise, optionalPromise)
}

func CatchRequiredSemaphore[SUPPLY any, REQUIRED any](
	semaphore *Semaphore,
	onRejected RejectedListener[SUPPLY],
	requiredPromise []*Promise[REQUIRED],
) *Promise[SUPPLY] {
	return CatchAllSemaphore[SUPPLY, REQUIRED, any](semaphore, onRejected, requiredPromise, nil)
}

func CatchRequired[SUPPLY any, REQUIRED any](
	onRejected RejectedListener[SUPPLY],
	requiredPromise []*Promise[REQUIRED],
) *Promise[SUPPLY] {
	return CatchRequiredSemaphore(nil, onRejected, requiredPromise)
}

func CatchSemaphore[SUPPLY any, NEED any](
	promise *Promise[NEED],
	semaphore *Semaphore,
	onRejected RejectedListener[SUPPLY],
) *Promise[SUPPLY] {
	return CatchRequiredSemaphore(semaphore, onRejected, []*Promise[NEED]{promise})
}

func Catch[SUPPLY any, NEED any](
	promise *Promise[NEED],
	onRejected RejectedListener[SUPPLY],
) *Promise[SUPPLY] {
	return CatchSemaphore(promise, nil, onRejected)
}

func ForCancelAllSemaphore[SUPPLY any, REQUIRED any, OPTIONAL any](
	semaphore *Semaphore,
	onCancelled CancelledListener,
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
) *Promise[SUPPLY] {
	return dependOn[REQUIRED, OPTIONAL, SUPPLY, any](
		semaphore,
		requiredPromise,
		optionalPromise,
		nil,
		nil,
		nil,
		&onCancelled,
	)
}

func ForCancelAll[SUPPLY any, REQUIRED any, OPTIONAL any](
	onCancelled CancelledListener,
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
) *Promise[SUPPLY] {
	return ForCancelAllSemaphore[SUPPLY, REQUIRED, OPTIONAL](nil, onCancelled, requiredPromise, optionalPromise)
}

func ForCancelRequiredSemaphore[SUPPLY any, REQUIRED any](
	semaphore *Semaphore,
	onCancelled CancelledListener,
	requiredPromise []*Promise[REQUIRED],
) *Promise[SUPPLY] {
	return ForCancelAllSemaphore[SUPPLY, REQUIRED, any](semaphore, onCancelled, requiredPromise, nil)
}

func ForCancelRequired[SUPPLY any, REQUIRED any](
	onCancelled CancelledListener,
	requiredPromise []*Promise[REQUIRED],
) *Promise[SUPPLY] {
	return ForCancelRequiredSemaphore[SUPPLY, REQUIRED](nil, onCancelled, requiredPromise)
}

func ForCancelSemaphore[SUPPLY any, NEED any](
	promise *Promise[NEED],
	semaphore *Semaphore,
	onCancelled CancelledListener,
) *Promise[SUPPLY] {
	return ForCancelRequiredSemaphore[SUPPLY, NEED](semaphore, onCancelled, []*Promise[NEED]{promise})
}

func ForCancel[SUPPLY any, NEED any](
	promise *Promise[NEED],
	onCancelled CancelledListener,
) *Promise[SUPPLY] {
	return ForCancelSemaphore[SUPPLY, NEED](promise, nil, onCancelled)
}

func FinallyAllSemaphore[SUPPLY any, T any, REQUIRED any, OPTIONAL any](
	semaphore *Semaphore,
	onFinally SettledListener[T],
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
) *Promise[SUPPLY] {
	return dependOn[REQUIRED, OPTIONAL, SUPPLY, T](
		semaphore,
		requiredPromise,
		optionalPromise,
		nil,
		nil,
		&onFinally,
		nil,
	)
}

func FinallyAll[SUPPLY any, T any, REQUIRED any, OPTIONAL any](
	onFinally SettledListener[T],
	requiredPromise []*Promise[REQUIRED],
	optionalPromise []*Promise[OPTIONAL],
) *Promise[SUPPLY] {
	return FinallyAllSemaphore[SUPPLY, T, REQUIRED, OPTIONAL](nil, onFinally, requiredPromise, optionalPromise)
}

func FinallyRequiredSemaphore[SUPPLY any, T any, REQUIRED any](
	semaphore *Semaphore,
	onFinally SettledListener[T],
	requiredPromise []*Promise[REQUIRED],
) *Promise[SUPPLY] {
	return FinallyAllSemaphore[SUPPLY, T, REQUIRED, any](semaphore, onFinally, requiredPromise, nil)
}

func FinallyRequired[SUPPLY any, T any, REQUIRED any](
	onFinally SettledListener[T],
	requiredPromise []*Promise[REQUIRED],
) *Promise[SUPPLY] {
	return FinallyRequiredSemaphore[SUPPLY, T, REQUIRED](nil, onFinally, requiredPromise)
}

func FinallySemaphore[SUPPLY any, T any, NEED any](
	promise *Promise[NEED],
	semaphore *Semaphore,
	onFinally SettledListener[T],
) *Promise[SUPPLY] {
	return FinallyRequiredSemaphore[SUPPLY, T, NEED](semaphore, onFinally, []*Promise[NEED]{promise})
}

func Finally[SUPPLY any, T any, NEED any](
	promise *Promise[NEED],
	onFinally SettledListener[T],
) *Promise[SUPPLY] {
	return FinallySemaphore[SUPPLY, T, NEED](promise, nil, onFinally)
}

func Resolve[T any](value *T) *Promise[T] {
	return (&Promise[T]{}).init(false, false).succeed(value)
}

func Reject[SUPPLY any](reason error) *Promise[SUPPLY] {
	return (&Promise[SUPPLY]{}).init(false, false).fail(reason)
}

func Cancelled[SUPPLY any]() *Promise[SUPPLY] {
	promise := (&Promise[SUPPLY]{}).init(false, false)
	promise.Cancel()
	return promise
}

func NewPromise[T any](job Job[T]) *Promise[T] {
	return (&Promise[T]{Job: job}).Init()
}

func NewPromiseWithSemaphore[T any](job Job[T], semaphore *Semaphore) *Promise[T] {
	return (&Promise[T]{Job: job, Semaphore: semaphore}).Init()
}
