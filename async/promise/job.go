package promise

type Job[T any] struct {
	Do func(resolver Resolver[T], rejector Rejector)
}
