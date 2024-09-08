//go:build ignore

package main

import (
	"fmt"
	"os"
)

var intSizes = []uint{8, 16, 32, 64}
var floatSizes = []int{32, 64}
var complexSizes = []int{64, 128}

const header = `package encoding

import (
	"io"
	"math"
	"reflect"
)

`

const intTemplate = `
type uint%[1]dBlueprint struct{}

func (i uint%[1]dBlueprint) from(bytes []byte, value reflect.Value) {
	value.SetUint(uint64(toUint%[1]d(bytes)))
}

func (i uint%[1]dBlueprint) to(value reflect.Value, bytes []byte) {
	fromUint%[1]d(uint%[1]d(value.Uint()), bytes)
}

func (i uint%[1]dBlueprint) size() uint {
	return %[2]d
}

func (b uint%[1]dBlueprint) describe(builder io.StringWriter) error {
	_, err := builder.WriteString("uint(%[1]d)")
	return err
}

type int%[1]dBlueprint struct{}


func (i int%[1]dBlueprint) from(bytes []byte, value reflect.Value) {
	value.SetInt(int64(int%[1]d(toUint%[1]d(bytes))))
}

func (i int%[1]dBlueprint) to(value reflect.Value, bytes []byte) {
	fromUint%[1]d(uint%[1]d(int%[1]d(value.Int())), bytes)
}

func (i int%[1]dBlueprint) size() uint {
	return %[2]d
}

func (b int%[1]dBlueprint) describe(builder io.StringWriter) error {
	_, err := builder.WriteString("int(%[1]d)")
	return err
}
`

const floatTemplate = `
type float%[1]dBlueprint struct{}

func (f float%[1]dBlueprint) from(bytes []byte, value reflect.Value) {
	value.SetFloat(float64(math.Float%[1]dfrombits(toUint%[1]d(bytes))))
}

func (f float%[1]dBlueprint) to(value reflect.Value, bytes []byte) {
	fromUint%[1]d(math.Float%[1]dbits(float%[1]d(value.Float())), bytes)
}

func (f float%[1]dBlueprint) size() uint {
	return %[2]d
}

func (f float%[1]dBlueprint) describe(builder io.StringWriter) error {
	_, err := builder.WriteString("float(%[1]d)")
	return err
}
`

const complexTemplate = `
type complex%[1]dBlueprint struct{}

func (c complex%[1]dBlueprint) from(bytes []byte, value reflect.Value) {
	realPart := math.Float%[3]dfrombits(toUint%[3]d(bytes))
	imagPart := math.Float%[3]dfrombits(toUint%[3]d(bytes[%[2]d/2:]))
	value.SetComplex(complex128(complex(realPart, imagPart)))
}

func (c complex%[1]dBlueprint) to(value reflect.Value, bytes []byte) {
	casted := complex%[1]d(value.Complex())
	fromUint%[3]d(math.Float%[3]dbits(real(casted)), bytes)
	fromUint%[3]d(math.Float%[3]dbits(imag(casted)), bytes[%[2]d/2:])
}

func (c complex%[1]dBlueprint) size() uint {
	return %[2]d
}

func (c complex%[1]dBlueprint) describe(builder io.StringWriter) error {
	_, err := builder.WriteString("complex(%[1]d)")
	return err
}
`

func handleWriteError(err error) {
	if err != nil {
		fmt.Printf("error occured when writing to schema_gen.go file: %s", err)
		os.Exit(1)
	}
}

func main() {
	file, err := os.Create("schema_gen.go")
	if err != nil {
		fmt.Printf("error occured when creating schema_gen.go file: %s", err)
		os.Exit(1)
	}
	_, err = file.WriteString(header)
	handleWriteError(err)
	for _, size := range intSizes {
		generateConvertFunctions(file, size)
	}
	for _, size := range intSizes {
		sizeInBytes := size / 8
		_, err := file.WriteString(fmt.Sprintf(intTemplate, size, sizeInBytes))
		handleWriteError(err)
	}
	for _, size := range floatSizes {
		sizeInBytes := size / 8
		_, err := file.WriteString(fmt.Sprintf(floatTemplate, size, sizeInBytes))
		handleWriteError(err)
	}
	for _, size := range complexSizes {
		sizeInBytes := size / 8
		floatSize := size / 2
		_, err := file.WriteString(fmt.Sprintf(complexTemplate, size, sizeInBytes, floatSize))
		handleWriteError(err)
	}
}

func generateConvertFunctions(file *os.File, bitSize uint) {
	if bitSize%8 != 0 || bitSize == 0 {
		panic("byteSize must be dividable by 8 and non-zero")
	}
	byteSize := bitSize / 8

	// from uint to bytes
	_, err := file.WriteString(fmt.Sprintf("func fromUint%[1]d(value uint%[1]d, dest []byte) {\n", bitSize))
	handleWriteError(err)

	// we last byte need no shifting, generate only for first byteSize-1 bytes
	for currentByte := uint(0); currentByte < byteSize-1; currentByte++ {
		shift := bitSize - (currentByte+1)*8
		_, err = file.WriteString(fmt.Sprintf("\tdest[%d] = byte(value >> %d)\n", currentByte, shift))
		handleWriteError(err)
	}
	_, err = file.WriteString(fmt.Sprintf("\tdest[%d] = byte(value)\n", byteSize-1))
	handleWriteError(err)
	_, err = file.WriteString("}\n\n")
	handleWriteError(err)

	// from bytes to uint
	_, err = file.WriteString(fmt.Sprintf("func toUint%[1]d(src []byte) uint%[1]d {\n", bitSize))
	handleWriteError(err)
	if byteSize > 1 {
		// size check for following access to src
		_, err = file.WriteString(fmt.Sprintf("\t_ = src[%d]\n", byteSize-1))
		handleWriteError(err)
	}
	_, err = file.WriteString("\treturn ")
	handleWriteError(err)
	// we last byte need no shifting, generate only for first byteSize-1 bytes
	for currentByte := uint(0); currentByte < byteSize-1; currentByte++ {
		shift := bitSize - (currentByte+1)*8
		_, err = file.WriteString(fmt.Sprintf("uint%d(src[%d]) << %d | ", bitSize, currentByte, shift))
		handleWriteError(err)
	}
	_, err = file.WriteString(fmt.Sprintf("uint%d(src[%d])\n", bitSize, byteSize-1))
	handleWriteError(err)
	_, err = file.WriteString("}\n\n")
	handleWriteError(err)
}
