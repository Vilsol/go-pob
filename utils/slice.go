package utils

func CastSlice[T any](s []interface{}) []T {
	out := make([]T, len(s))
	for i, v := range s {
		out[i] = v.(T)
	}
	return out
}
