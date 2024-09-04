//go:generate go run ./gen/numbers.go

package encoding

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

const pointerSize uint = 1

type arrayBlueprint struct {
	length  uint
	element blueprint
}

func (a arrayBlueprint) from(bytes []byte, value reflect.Value) {
	var offset uint
	for i := uint(0); i < a.length; i++ {
		a.element.from(bytes[offset:], value.Index(int(i)))
		offset += a.element.size()
	}
}

func (a arrayBlueprint) to(value reflect.Value, bytes []byte) {
	var offset uint
	for i := uint(0); i < a.length; i++ {
		a.element.to(value.Index(int(i)), bytes[offset:])
		offset += a.element.size()
	}
}

func (a arrayBlueprint) size() uint {
	return a.element.size() * a.length
}

type boolBlueprint struct {
}

func (b boolBlueprint) from(bytes []byte, value reflect.Value) {
	value.SetBool(bytes[0] != 0)
	return
}

func (b boolBlueprint) to(value reflect.Value, bytes []byte) {
	bytes[0] = 0
	if value.Bool() {
		bytes[0] = 1
	}
}

func (b boolBlueprint) size() uint {
	return 1
}

type sliceBlueprint struct {
	length  uint32
	element blueprint
}

func (s sliceBlueprint) from(bytes []byte, value reflect.Value) {
	realLength := toUint32(bytes)
	bytes = bytes[4:] //
	value.Grow(int(realLength))
	value.SetLen(int(realLength))
	var offset uint
	for i := 0; i < int(realLength); i++ {
		s.element.from(bytes[4+offset:], value.Index(i))
		offset += s.element.size()
	}
}

func (s sliceBlueprint) to(value reflect.Value, dest []byte) {
	persistedLen := uint32(min(uint(value.Len()), uint(s.length)))
	var offset uint
	for i := 0; i < int(persistedLen); i++ {
		s.element.to(value.Index(i), dest[4+offset:])
		offset += s.element.size()
	}
	fromUint32(persistedLen, dest)
}

func (s sliceBlueprint) size() uint {
	return uint(s.length) * s.element.size()
}

type stringBlueprint struct {
	length uint32 // length in bytes
}

func (s stringBlueprint) from(bytes []byte, value reflect.Value) {
	realLength := toUint32(bytes)
	bytes = bytes[4:]
	builder := strings.Builder{}
	builder.Grow(int(realLength))
	for i := uint32(0); i < realLength; i++ {
		_ = builder.WriteByte(bytes[i]) // error never occur
	}
	value.SetString(builder.String())
}

func (s stringBlueprint) to(value reflect.Value, dest []byte) {
	reader := strings.NewReader(value.String())
	var written uint32 = 0
	for {
		runeToBeWritten, i, err := reader.ReadRune()
		if errors.Is(err, io.EOF) || written+uint32(i) > s.length {
			break
		}
		if err != nil {
			// should not happen
			panic(fmt.Sprintf("unknown error %s", err))
		}
		// first 4 bytes are for length
		written += uint32(utf8.EncodeRune(dest[4+written:], runeToBeWritten))
	}
	fromUint32(written, dest)
}

func (s stringBlueprint) size() uint {
	return uint(s.length)
}

type structField struct {
	blueprint
	fieldIndex int
}

type structBlueprint struct {
	structType reflect.Type
	fields     []structField
	totalSize  uint
}

func (s structBlueprint) from(bytes []byte, value reflect.Value) {
	var offset uint
	for _, field := range s.fields {
		field.from(bytes[offset:], value.Field(field.fieldIndex))
		offset += field.size()
	}
}

func (s structBlueprint) to(value reflect.Value, bytes []byte) {
	var offset uint
	for _, field := range s.fields {
		field.to(value.Field(field.fieldIndex), bytes[offset:])
		offset += field.size()
	}
}

func (s structBlueprint) size() uint {
	return s.totalSize
}

const nilPointer byte = 0

type pointerBlueprint struct {
	childSize uint
	element   blueprint
}

func (p pointerBlueprint) size() uint {
	return p.childSize + pointerSize
}

func (p pointerBlueprint) from(bytes []byte, value reflect.Value) {
	if bytes[0] == nilPointer {
		value.SetZero()
		return
	}
	// TODO: handle possible nil pointer
	p.element.from(bytes[pointerSize:], value.Elem())
}

func (p pointerBlueprint) to(value reflect.Value, dest []byte) {
	if value.IsNil() {
		dest[0] = nilPointer
	}
	p.element.to(value.Elem(), dest[pointerSize:])
}

type blueprint interface {
	from([]byte, reflect.Value)
	to(reflect.Value, []byte)
	size() uint
}

type Serializer[T any] struct {
	blueprint blueprint
	size      uint
}

func (s Serializer[T]) Size() uint {
	return s.blueprint.size()
}

func (s Serializer[T]) Encode(value T) []byte {
	var data = make([]byte, s.size)
	s.blueprint.to(reflect.ValueOf(value), data)
	return data
}

func (s Serializer[T]) Decode(bytes []byte) T {
	var value T
	s.blueprint.from(bytes, reflect.ValueOf(&value))
	return value
}
