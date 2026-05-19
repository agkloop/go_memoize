package serializers

import (
	"bytes"
	"encoding/gob"
)

type Gob[V any] struct{}

func (Gob[V]) Marshal(value V) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(value)
	return buf.Bytes(), err
}

func (Gob[V]) Unmarshal(data []byte) (V, error) {
	var value V
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&value)
	return value, err
}
