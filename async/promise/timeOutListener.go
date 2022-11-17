package promise

import "time"

type TimeOutListener struct {
	OnTimeOut func(duration time.Duration)
}
