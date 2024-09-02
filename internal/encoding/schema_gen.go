package encoding

import (
	"math"
	"reflect"
)

func fromUint8(value uint8, dest []byte) {
	dest[0] = byte(value)
}

func toUint8(src []byte) uint8 {
	return uint8(src[0])
}

func fromUint16(value uint16, dest []byte) {
	dest[0] = byte(value >> 8)
	dest[1] = byte(value)
}

func toUint16(src []byte) uint16 {
	_ = src[1]
	return uint16(src[0]) << 8 | uint16(src[1])
}

func fromUint32(value uint32, dest []byte) {
	dest[0] = byte(value >> 24)
	dest[1] = byte(value >> 16)
	dest[2] = byte(value >> 8)
	dest[3] = byte(value)
}

func toUint32(src []byte) uint32 {
	_ = src[3]
	return uint32(src[0]) << 24 | uint32(src[1]) << 16 | uint32(src[2]) << 8 | uint32(src[3])
}

func fromUint64(value uint64, dest []byte) {
	dest[0] = byte(value >> 56)
	dest[1] = byte(value >> 48)
	dest[2] = byte(value >> 40)
	dest[3] = byte(value >> 32)
	dest[4] = byte(value >> 24)
	dest[5] = byte(value >> 16)
	dest[6] = byte(value >> 8)
	dest[7] = byte(value)
}

func toUint64(src []byte) uint64 {
	_ = src[7]
	return uint64(src[0]) << 56 | uint64(src[1]) << 48 | uint64(src[2]) << 40 | uint64(src[3]) << 32 | uint64(src[4]) << 24 | uint64(src[5]) << 16 | uint64(src[6]) << 8 | uint64(src[7])
}


type uint8Blueprint struct{}

func (i uint8Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetUint(uint64(toUint8(bytes)))
	return nil
}

func (i uint8Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint8(uint8(value.Uint()), bytes)
	return nil
}

func (i uint8Blueprint) size() uint {
	return 1
}

type int8Blueprint struct{}


func (i int8Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetInt(int64(int8(toUint8(bytes))))
	return nil
}

func (i int8Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint8(uint8(int8(value.Int())), bytes)
	return nil
}

func (i int8Blueprint) size() uint {
	return 1
}

type uint16Blueprint struct{}

func (i uint16Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetUint(uint64(toUint16(bytes)))
	return nil
}

func (i uint16Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint16(uint16(value.Uint()), bytes)
	return nil
}

func (i uint16Blueprint) size() uint {
	return 2
}

type int16Blueprint struct{}


func (i int16Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetInt(int64(int16(toUint16(bytes))))
	return nil
}

func (i int16Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint16(uint16(int16(value.Int())), bytes)
	return nil
}

func (i int16Blueprint) size() uint {
	return 2
}

type uint32Blueprint struct{}

func (i uint32Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetUint(uint64(toUint32(bytes)))
	return nil
}

func (i uint32Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint32(uint32(value.Uint()), bytes)
	return nil
}

func (i uint32Blueprint) size() uint {
	return 4
}

type int32Blueprint struct{}


func (i int32Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetInt(int64(int32(toUint32(bytes))))
	return nil
}

func (i int32Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint32(uint32(int32(value.Int())), bytes)
	return nil
}

func (i int32Blueprint) size() uint {
	return 4
}

type uint64Blueprint struct{}

func (i uint64Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetUint(uint64(toUint64(bytes)))
	return nil
}

func (i uint64Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint64(uint64(value.Uint()), bytes)
	return nil
}

func (i uint64Blueprint) size() uint {
	return 8
}

type int64Blueprint struct{}


func (i int64Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetInt(int64(int64(toUint64(bytes))))
	return nil
}

func (i int64Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint64(uint64(int64(value.Int())), bytes)
	return nil
}

func (i int64Blueprint) size() uint {
	return 8
}

type float32Blueprint struct{}

func (f float32Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetFloat(float64(math.Float32frombits(toUint32(bytes))))
	return nil
}

func (f float32Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint32(math.Float32bits(float32(value.Float())), bytes)
	return nil
}

func (f float32Blueprint) size() uint {
	return 4
}

type float64Blueprint struct{}

func (f float64Blueprint) from(bytes []byte, value reflect.Value) error {
	value.SetFloat(float64(math.Float64frombits(toUint64(bytes))))
	return nil
}

func (f float64Blueprint) to(value reflect.Value, bytes []byte) error {
	fromUint64(math.Float64bits(float64(value.Float())), bytes)
	return nil
}

func (f float64Blueprint) size() uint {
	return 8
}

type complex64Blueprint struct{}

func (c complex64Blueprint) from(bytes []byte, value reflect.Value) error {
	realPart := math.Float32frombits(toUint32(bytes))
	imagPart := math.Float32frombits(toUint32(bytes[8/2:]))
	value.SetComplex(complex128(complex(realPart, imagPart)))
	return nil
}

func (c complex64Blueprint) to(value reflect.Value, bytes []byte) error {
	casted := complex64(value.Complex())
	fromUint32(math.Float32bits(real(casted)), bytes)
	fromUint32(math.Float32bits(imag(casted)), bytes[8/2:])
	return nil
}

func (c complex64Blueprint) size() uint {
	return 8
}

type complex128Blueprint struct{}

func (c complex128Blueprint) from(bytes []byte, value reflect.Value) error {
	realPart := math.Float64frombits(toUint64(bytes))
	imagPart := math.Float64frombits(toUint64(bytes[16/2:]))
	value.SetComplex(complex128(complex(realPart, imagPart)))
	return nil
}

func (c complex128Blueprint) to(value reflect.Value, bytes []byte) error {
	casted := complex128(value.Complex())
	fromUint64(math.Float64bits(real(casted)), bytes)
	fromUint64(math.Float64bits(imag(casted)), bytes[16/2:])
	return nil
}

func (c complex128Blueprint) size() uint {
	return 16
}
