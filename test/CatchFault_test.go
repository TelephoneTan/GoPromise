package test

import (
	"github.com/TelephoneTan/GoPromise/async/promise"
	"testing"
	"unsafe"
)

func TestCatchFault(t *testing.T) {
	trigger := promise.NewPromise(promise.Job[any]{
		Do: func(rs promise.Resolver[any], re promise.Rejector) {
			b := make([]byte, 1)
			println("access some memory")
			foo := (*int)(unsafe.Pointer(uintptr(unsafe.Pointer(&b[0])) + uintptr(999999999)))
			println(*foo + 1)
		},
	})
	promise.Catch(trigger, promise.RejectedListener[any]{
		OnRejected: func(reason error) any {
			println("catch:", reason.Error())
			return nil
		},
	}).Await()
	println("end")
}
