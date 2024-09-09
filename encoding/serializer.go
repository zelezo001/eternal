package encoding

import (
	"bytes"
	"crypto/sha512"
	"reflect"
)

type Serializer[T any] struct {
	blueprint blueprint
	size      uint
}

func (s Serializer[T]) Size() uint {
	return s.blueprint.size()
}

func (s Serializer[T]) Serialize(value T) []byte {
	var data = make([]byte, s.size)
	s.blueprint.to(reflect.ValueOf(value), data)
	return data
}

func (s Serializer[T]) Deserialize(bytes []byte) T {
	var value T
	s.blueprint.from(bytes, reflect.ValueOf(&value).Elem())
	return value
}

func (s Serializer[T]) Signature() [64]byte {
	var builder = &bytes.Buffer{}
	err := s.blueprint.describe(builder)
	if err != nil {
		// bytes.Buffer does not produce err on writes
		panic(err)
	}

	return sha512.Sum512(builder.Bytes())
}
