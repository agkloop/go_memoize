package serializers

import "encoding/json"

type JSON[V any] struct{}

func (JSON[V]) Marshal(value V) ([]byte, error) {
	return json.Marshal(value)
}

func (JSON[V]) Unmarshal(data []byte) (V, error) {
	var value V
	err := json.Unmarshal(data, &value)
	return value, err
}
