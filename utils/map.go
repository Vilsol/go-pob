package utils

import (
	"github.com/rs/zerolog/log"
)

func Has[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}

func HasTrue[K comparable, V any](m map[K]V, key K) bool {
	val, ok := m[key]
	if !ok {
		return false
	}

	if boolVal, ok := getBool(val); ok {
		return boolVal
	}

	log.Warn().
		Interface("key", key).
		Interface("value", val).
		Msg("failed to cast to bool")

	return false
}

func getBool(value interface{}) (bool, bool) {
	val, ok := value.(bool)
	return val, ok
}

func GetOr[V any, K comparable](m map[K]V, key K, or V) V {
	if val, ok := m[key]; ok {
		return val
	}
	return or
}

func MissingOrFalse[K comparable, V any](m map[K]V, key K) bool {
	val, ok := m[key]
	if !ok {
		return true
	}

	if boolVal, ok := getBool(val); ok {
		return boolVal
	}

	log.Warn().
		Interface("key", key).
		Interface("value", val).
		Msg("failed to cast to bool")

	return false
}
