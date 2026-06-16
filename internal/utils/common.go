package utils

func SafeRef[T any](ref T) *T {
	return &ref
}

func SafeDeref[T any](ref *T) T {
	if ref == nil {
		return *new(T)
	}
	return *ref
}
