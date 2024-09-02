package encoding

import (
	"errors"
	"fmt"
	"math/bits"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type context struct {
	seenStructTypes map[reflect.Type]struct{} // prevents recursive definitions
}

var ErrUnsupportedType = errors.New("provided type is not supported")
var ErrLengthMustBeSet = errors.New("cannot determine ")

var ErrInvalidAnnotation = errors.New("tag had invalid format")
var ErrRecursiveStructDefinition = errors.New("struct cannot contain itself")

var parsedStructs sync.Map

type Primitive interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~complex64 | ~complex128
}

func CreateForPrimitive[T Primitive]() (Serializer[T], error) {
	blueprint, size, err := handleType(context{}, reflect.TypeFor[T](), config{})
	return Serializer[T]{
		size:      size,
		blueprint: blueprint,
	}, err
}

func CreateForString[T ~string](maxLength uint32) Serializer[T] {
	blueprint, size, err := handleType(context{}, reflect.TypeFor[T](), config{length: maxLength})
	if err != nil {
		// handleType() should not return errors for string
		panic(err)
	}

	return Serializer[T]{
		size:      size,
		blueprint: blueprint,
	}
}

func CreateForStringSlice[T interface{ ~[]E }, E interface{ ~string | ~*string }](
	maxLength uint32, maxStringLength uint32,
) (Serializer[T], error) {
	config := config{length: maxLength, elementLength: maxStringLength}
	blueprint, size, err := handleType(context{}, reflect.TypeFor[T](), config)
	return Serializer[T]{
		size:      size,
		blueprint: blueprint,
	}, err
}

func CreateForSlice[T interface{ ~[]E }, E any](maxLength uint32) (Serializer[T], error) {
	blueprint, size, err := handleType(context{}, reflect.TypeFor[T](), config{length: maxLength})
	return Serializer[T]{
		size:      size,
		blueprint: blueprint,
	}, err
}

func CreateForStruct[T any]() (Serializer[T], error) {
	blueprint, size, err := handleType(context{}, reflect.TypeFor[T](), config{})
	if err != nil {
		return Serializer[T]{}, err
	}

	return Serializer[T]{
		size:      size,
		blueprint: blueprint,
	}, nil
}

type config struct {
	length, elementLength uint32
	ignore                bool // only used for struct fields
}

const separator = ";"
const valueSeparator = ":"
const tagName = "eternal"

func parseConfig(raw string) (config, error) {
	var config config
	for _, property := range strings.Split(raw, separator) {
		var split = strings.SplitN(property, valueSeparator, 2)
		switch property := strings.ToLower(strings.TrimSpace(split[0])); property {
		case "size":
			if len(split) == 1 {
				return config, fmt.Errorf("%w: property size must have value set", ErrInvalidAnnotation)
			}
			sizeString := strings.TrimSpace(split[1])
			size, err := strconv.ParseUint(sizeString, 10, 32)
			if err != nil {
				return config, fmt.Errorf("%w: property size be of type uint: %w", ErrInvalidAnnotation, err)
			}
			config.length = uint32(size)
		case "elementSize":
			if len(split) == 1 {
				return config, fmt.Errorf("%w: property elementSize must have value set", ErrInvalidAnnotation)
			}
			sizeString := strings.TrimSpace(split[1])
			size, err := strconv.ParseUint(sizeString, 10, 32)
			if err != nil {
				return config, fmt.Errorf("%w: property elementSize be of type uint: %w", ErrInvalidAnnotation, err)
			}
			config.elementLength = uint32(size)
		case "ignored":
			config.ignore = true
		case "":
			continue
		default:
			return config, fmt.Errorf("%w: unkown property %s", ErrInvalidAnnotation, property)
		}
	}

	return config, nil
}

func handleType(ctx context, _type reflect.Type, blueprintConfig config) (blueprint, uint, error) {
	switch _type.Kind() {
	case reflect.Bool:
		return boolBlueprint{}, 1, nil
	case reflect.Int:
		switch bits.UintSize {
		case 32:
			return int32Blueprint{}, 4, nil
		case 64:
			return int64Blueprint{}, 8, nil
		default:
			panic("int must be 32 or 64 bits")
		}
	case reflect.Int8:
		return int8Blueprint{}, 1, nil
	case reflect.Int16:
		return int16Blueprint{}, 2, nil
	case reflect.Int32:
		return int32Blueprint{}, 4, nil
	case reflect.Int64:
		return int64Blueprint{}, 8, nil
	case reflect.Uint:
		switch bits.UintSize {
		case 32:
			return uint32Blueprint{}, 4, nil
		case 64:
			return uint64Blueprint{}, 8, nil
		default:
			panic("int must be 32 or 64 bits")
		}
	case reflect.Uint8:
		return uint8Blueprint{}, 1, nil
	case reflect.Uint16:
		return uint16Blueprint{}, 2, nil
	case reflect.Uint32:
		return uint32Blueprint{}, 4, nil
	case reflect.Uint64:
		return uint64Blueprint{}, 8, nil
	case reflect.Float32:
		return float32Blueprint{}, 4, nil
	case reflect.Float64:
		return float64Blueprint{}, 8, nil
	case reflect.Complex64:
		return complex64Blueprint{}, 8, nil
	case reflect.Complex128:
		return complex128Blueprint{}, 16, nil
	case reflect.Pointer:
		inner, size, err := handleType(ctx, _type.Elem(), blueprintConfig)
		return pointerBlueprint{
			element:   inner,
			childSize: size,
		}, size + pointerSize, err
	case reflect.Array:
		inner, size, err := handleType(ctx, _type.Elem(), config{
			length: blueprintConfig.elementLength,
		})
		return arrayBlueprint{
			element: inner,
			length:  uint(_type.Len()), // overflow cannot happen as uint has more positive values than int
		}, size + pointerSize, err
	case reflect.Slice:
		if blueprintConfig.length != 0 {
			inner, size, err := handleType(ctx, _type.Elem(), config{
				length: blueprintConfig.elementLength,
			})
			return sliceBlueprint{
				element: inner,
				length:  blueprintConfig.length,
			}, size + pointerSize, err
		}
		return nil, 0, ErrLengthMustBeSet
	case reflect.String:
		if blueprintConfig.length != 0 {
			return stringBlueprint{length: blueprintConfig.length}, uint(blueprintConfig.length), nil
		}
		return nil, 0, ErrLengthMustBeSet
	case reflect.Struct:
		if _, ok := ctx.seenStructTypes[_type]; ok {
			return nil, 0, fmt.Errorf("could not handle type %s: %w", _type, ErrRecursiveStructDefinition)
		}
		if blueprint, found := parsedStructs.Load(_type); found {
			typed := blueprint.(structBlueprint)
			return typed, typed.totalSize, nil
		}
		blueprint := structBlueprint{
			structType: _type,
			fields:     make([]structField, 0, _type.NumField()),
		}
		var size uint
		ctx.seenStructTypes[_type] = struct{}{}
		defer delete(ctx.seenStructTypes, _type)
		for i := 0; i < _type.NumField(); i++ {
			field := _type.Field(i)
			config, err := parseConfig(field.Tag.Get(tagName))
			if err != nil {
				return nil, 0, fmt.Errorf("could not parse config for property %s of type %s: %w", field.Name, _type,
					err)
			}
			if config.ignore {
				continue
			}
			fieldBlueprint, fieldSize, err := handleType(ctx, field.Type, config)
			if err != nil {
				return nil, 0, fmt.Errorf("could not parse config for property %s of type %s: %w", field.Name, _type,
					err)
			}
			size += fieldSize
			blueprint.fields = append(blueprint.fields, structField{
				blueprint:  fieldBlueprint,
				fieldIndex: i,
			})
		}
		blueprint.fields = slices.Clip(blueprint.fields)
		blueprint.totalSize = size
		parsedStructs.Store(_type, blueprint)
		return blueprint, size, nil
	default:
		return nil, 0, fmt.Errorf("type %s: %w", _type.String(), ErrUnsupportedType)
	}
}
