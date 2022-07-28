package utils

func CopySlice[T any](s []T) []T {
	out := make([]T, len(s))
	copy(out, s)
	return out
}

func CopyMap[K comparable, V any](s map[K]V) map[K]V {
	out := make(map[K]V)
	for k, v := range s {
		out[k] = v
	}
	return out
}

func Ptr[T any](a T) *T {
	return &a
}
