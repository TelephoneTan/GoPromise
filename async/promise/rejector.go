package promise

type Rejector interface {
	cancel()
	Reject(e error)
}
