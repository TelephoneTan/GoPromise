package promise

type CancelledListener struct {
	OnCancelled func()
}
