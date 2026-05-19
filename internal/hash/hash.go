package hash

import (
	"fmt"
	"math"
)

const (
	Offset64 = uint64(14695981039346656037)
	Prime64  = uint64(1099511628211)
)

func String(h uint64, key string) uint64 {
	length := len(key)
	for i := 0; i < length/4*4; i += 4 {
		h = (h ^ uint64(key[i])) * Prime64
		h = (h ^ uint64(key[i+1])) * Prime64
		h = (h ^ uint64(key[i+2])) * Prime64
		h = (h ^ uint64(key[i+3])) * Prime64
	}
	for i := length / 4 * 4; i < length; i++ {
		h = (h ^ uint64(key[i])) * Prime64
	}
	return h
}

func Uint(h uint64, key uint64) uint64 {
	return (h ^ key) * Prime64
}

func Bool(h uint64, key bool) uint64 {
	if key {
		return (h ^ 1) * Prime64
	}
	return h * Prime64
}

func Comparable[K comparable](h uint64, key K) uint64 {
	switch v := any(key).(type) {
	case string:
		return String(h, v)
	case int:
		return Uint(h, uint64(v))
	case int8:
		return Uint(h, uint64(v))
	case int16:
		return Uint(h, uint64(v))
	case int32:
		return Uint(h, uint64(v))
	case int64:
		return Uint(h, uint64(v))
	case uint:
		return Uint(h, uint64(v))
	case uint8:
		return Uint(h, uint64(v))
	case uint16:
		return Uint(h, uint64(v))
	case uint32:
		return Uint(h, uint64(v))
	case uint64:
		return Uint(h, v)
	case uintptr:
		return Uint(h, uint64(v))
	case float32:
		return Uint(h, math.Float64bits(float64(v)))
	case float64:
		return Uint(h, math.Float64bits(v))
	case bool:
		return Bool(h, v)
	default:
		panic(fmt.Sprintf("unsupported type for caching %T", key))
	}
}
