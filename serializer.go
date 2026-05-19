package memoize

type Serializer[V any] interface {
	Marshal(V) ([]byte, error)
	Unmarshal([]byte) (V, error)
}
