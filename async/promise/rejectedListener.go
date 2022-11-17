package promise

type RejectedListener[SUPPLY any] struct {
	OnRejected func(reason error) any
}
