package serializers

type Func[V any] struct {
	MarshalFunc   func(V) ([]byte, error)
	UnmarshalFunc func([]byte) (V, error)
}

func (f Func[V]) Marshal(value V) ([]byte, error) {
	return f.MarshalFunc(value)
}

func (f Func[V]) Unmarshal(data []byte) (V, error) {
	return f.UnmarshalFunc(data)
}
