package test

import (
	"github.com/TelephoneTan/GoPromise/async/promise"
	"testing"
)

func TestCancelled(t *testing.T) {
	p := promise.Cancelled[string]()
	p = promise.Then(p, promise.FulfilledListener[string, string]{
		OnFulfilled: func(value *string) any {
			println("hello, world")
			return nil
		},
	})
	p = promise.ForCancel[string](p, promise.CancelledListener{
		OnCancelled: func() {
			println("cancelled")
		},
	})
	p.Await()
}
