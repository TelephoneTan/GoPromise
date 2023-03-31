package task

import (
	"errors"
	"github.com/TelephoneTan/GoPromise/async/promise"
	"sync"
	"sync/atomic"
	"time"
)

const archivedTip = "定时任务已归档"

type timedToken struct {
	delay *time.Duration
}

type Timed struct {
	interval       atomic.Pointer[time.Duration]
	lifeLimited    bool
	lifeTimes      atomic.Int64
	token          chan timedToken
	block          chan timedToken
	archived       line
	modifying      line
	succeededTimes atomic.Int64
	succeeded      chan int64
	failed         chan error
	cancelled      line
	promise        *promise.Promise[int64]
	start          sync.Once
	//
	Job       promise.Job[bool]
	Semaphore *promise.Semaphore
}

func (t *Timed) Init(interval time.Duration, times ...int64) *Timed {
	t.interval.Store(&interval)
	t.lifeLimited = len(times) > 0
	if t.lifeLimited {
		t.lifeTimes.Store(times[0])
	} else {
		t.lifeTimes.Store(0)
	}
	t.token = make(chan timedToken, 1)
	t.block = make(chan timedToken, 1)
	t.archived = make(line)
	t.modifying = make(line, 1)
	t.succeeded = make(chan int64, 1)
	t.failed = make(chan error, 1)
	t.cancelled = make(line, 1)
	t.promise = (&promise.Promise[int64]{
		Job: promise.Job[int64]{
			Do: func(rs promise.Resolver[int64], re promise.Rejector) {
				<-t.archived
				select {
				case st := <-t.succeeded:
					rs.ResolveValue(st)
				case e := <-t.failed:
					re.Reject(e)
				case <-t.cancelled:
				}
			},
		},
	}).Init()
	return t
}

func (t *Timed) IsArchived() bool {
	select {
	case <-t.archived:
		return true
	default:
		return false
	}
}

func modify[R any](timed *Timed, op promise.Job[R]) *promise.Promise[R] {
	operate := (&promise.Promise[R]{
		Job: promise.Job[R]{
			Do: func(rs promise.Resolver[R], re promise.Rejector) {
				timed.modifying <- signal
				if timed.IsArchived() {
					panic(archivedTip)
				} else {
					op.Do(rs, re)
				}
			},
		},
	}).Init()
	promise.Finally[R](operate, promise.SettledListener[any]{
		OnSettled: func() *promise.Promise[any] {
			<-timed.modifying
			return nil
		},
	})
	return operate
}

func (t *Timed) end() {
	modify(t, promise.Job[any]{
		Do: func(rs promise.Resolver[any], re promise.Rejector) {
			t.succeeded <- t.succeededTimes.Load()
			close(t.archived)
			rs.ResolveValue(nil)
		},
	})
}

func (t *Timed) error(e error) {
	modify(t, promise.Job[any]{
		Do: func(rs promise.Resolver[any], re promise.Rejector) {
			t.failed <- e
			close(t.archived)
			rs.ResolveValue(nil)
		},
	})
}

func (t *Timed) Cancel() *promise.Promise[any] {
	return modify(t, promise.Job[any]{
		Do: func(rs promise.Resolver[any], re promise.Rejector) {
			t.cancelled <- signal
			t.promise.Cancel()
			close(t.archived)
			rs.ResolveValue(nil)
		},
	})
}

func (t *Timed) checkAlive(consumeLife bool) *promise.Promise[any] {
	return modify(t, promise.Job[any]{
		Do: func(rs promise.Resolver[any], re promise.Rejector) {
			lt := t.lifeTimes.Load()
			alive := !t.lifeLimited || lt > 0
			if alive && consumeLife {
				lt--
			}
			t.lifeTimes.Store(lt)
			if !alive {
				t.end()
				panic(nil)
			} else {
				rs.ResolveValue(nil)
			}
		},
	})
}

func (t *Timed) prepare() *promise.Promise[any] {
	resume := (&promise.Promise[timedToken]{
		Job: promise.Job[timedToken]{
			Do: func(rs promise.Resolver[timedToken], re promise.Rejector) {
				select {
				case v := <-t.token:
					t.token <- timedToken{}
					rs.ResolveValue(v)
				case <-t.archived:
					panic(archivedTip)
				}
			},
		},
	}).Init()
	checkAlive := promise.Then(resume, promise.FulfilledListener[timedToken, timedToken]{
		OnFulfilled: func(token timedToken) any {
			checkAlive := t.checkAlive(false)
			return promise.Then(checkAlive, promise.FulfilledListener[any, timedToken]{
				OnFulfilled: func(_ any) any {
					return token
				},
			})
		},
	})
	delay := promise.Then(checkAlive, promise.FulfilledListener[timedToken, any]{
		OnFulfilled: func(token timedToken) any {
			if token.delay != nil {
				time.Sleep(*token.delay)
			}
			return nil
		},
	})
	return promise.Then(delay, promise.FulfilledListener[any, any]{
		OnFulfilled: func(_ any) any {
			return t.checkAlive(true)
		},
	})
}

func (t *Timed) run() {
	prepare := t.prepare()
	execute := promise.Then(prepare, promise.FulfilledListener[any, bool]{
		OnFulfilled: func(_ any) any {
			return (&promise.Promise[bool]{
				Job:       t.Job,
				Semaphore: t.Semaphore,
			}).Init()
		},
	})
	recordSuccess := promise.Then(execute, promise.FulfilledListener[bool, bool]{
		OnFulfilled: func(res bool) any {
			t.succeededTimes.Add(1)
			return res
		},
	})
	judgeResult := promise.Then(recordSuccess, promise.FulfilledListener[bool, any]{
		OnFulfilled: func(res bool) any {
			if !res {
				t.end()
				panic("提前结束了")
			}
			return nil
		},
	})
	checkAlive := promise.Then(judgeResult, promise.FulfilledListener[any, any]{
		OnFulfilled: func(_ any) any {
			return t.checkAlive(false)
		},
	})
	delay := promise.Then(checkAlive, promise.FulfilledListener[any, any]{
		OnFulfilled: func(_ any) any {
			d := t.interval.Load()
			if d != nil {
				time.Sleep(*d)
			}
			return nil
		},
	})
	checkAlive = promise.Then(delay, promise.FulfilledListener[any, any]{
		OnFulfilled: func(_ any) any {
			return t.checkAlive(false)
		},
	})
	nextTask := promise.Then(checkAlive, promise.FulfilledListener[any, any]{
		OnFulfilled: func(_ any) any {
			t.run()
			return nil
		},
	})
	promise.Catch(nextTask, promise.RejectedListener[any]{
		OnRejected: func(reason error) any {
			t.error(reason)
			return nil
		},
	})
}

func (t *Timed) Pause() *promise.Promise[any] {
	return modify(t, promise.Job[any]{
		Do: func(rs promise.Resolver[any], re promise.Rejector) {
			select {
			case v := <-t.token:
				t.block <- v
				rs.ResolveValue(nil)
			default:
				panic("定时任务当前处于不可暂停状态，请稍后重试")
			}
		},
	})
}

func (t *Timed) Resume(delay ...time.Duration) *promise.Promise[any] {
	return modify(t, promise.Job[any]{
		Do: func(rs promise.Resolver[any], re promise.Rejector) {
			select {
			case v := <-t.block:
				if len(delay) > 0 {
					d := delay[0]
					v.delay = &d
				} else {
					v.delay = nil
				}
				t.token <- v
				rs.ResolveValue(nil)
			default:
				panic("定时任务非暂停状态，不能恢复")
			}
		},
	})
}

func (t *Timed) SetInterval(interval time.Duration) *promise.Promise[any] {
	return modify(t, promise.Job[any]{
		Do: func(rs promise.Resolver[any], re promise.Rejector) {
			t.interval.Store(&interval)
			rs.ResolveValue(nil)
		},
	})
}

func (t *Timed) AddTimesBy(delta int64) *promise.Promise[int64] {
	if !t.lifeLimited {
		return promise.Reject[int64](errors.New("该定时任务无固定运行次数"))
	} else if delta < 0 {
		return promise.Reject[int64](errors.New("增加的次数不能小于 0"))
	} else if delta == 0 {
		return promise.Resolve(delta)
	} else {
		return modify(t, promise.Job[int64]{
			Do: func(rs promise.Resolver[int64], re promise.Rejector) {
				t.lifeTimes.Add(delta)
				rs.ResolveValue(delta)
			},
		})
	}
}

func (t *Timed) ReduceTimeBy(delta int64) *promise.Promise[int64] {
	if !t.lifeLimited {
		return promise.Reject[int64](errors.New("该定时任务无固定运行次数"))
	} else if delta < 0 {
		return promise.Reject[int64](errors.New("减少的次数不能小于 0"))
	} else if delta == 0 {
		return promise.Resolve(delta)
	} else {
		return modify(t, promise.Job[int64]{
			Do: func(rs promise.Resolver[int64], re promise.Rejector) {
				current := t.lifeTimes.Load()
				rd := delta
				if current <= 0 {
					rd = 0
				} else if rd > current {
					rd = current
				}
				t.lifeTimes.Store(current - rd)
				rs.ResolveValue(rd)
			},
		})
	}
}

func (t *Timed) Start(delay ...time.Duration) *promise.Promise[int64] {
	t.start.Do(func() {
		start := (&promise.Promise[any]{
			Job: promise.Job[any]{
				Do: func(rs promise.Resolver[any], re promise.Rejector) {
					t.run()
					t.block <- timedToken{}
					rs.ResolvePromise(t.Resume(delay...))
				},
			},
		}).Init()
		promise.Catch(start, promise.RejectedListener[any]{
			OnRejected: func(reason error) any {
				t.error(reason)
				return nil
			},
		})
	})
	return t.promise
}
