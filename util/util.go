package util

func Assign[T any](target *T, value T) T {
	*target = value
	return value
}
