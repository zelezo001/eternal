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

var (
	ErrUnsupportedType           = errors.New("provided type is not supported")
	ErrLengthMustBeSet           = errors.New("cannot determine ")
	ErrInvalidAnnotation         = errors.New("tag had invalid format")
	ErrRecursiveStructDefinition = errors.New("struct cannot contain itself")
)

type Primitive interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 | ~complex64 | ~complex128 | ~bool
}

func CreateForPrimitive[T Primitive]() Serializer[T] {
	blueprint, err := handleType(newContext(), reflect.TypeFor[T](), config{})
	if err != nil {
		// handleType() should not return errors for primitive
		panic(err)
	}
	return Serializer[T]{
		size:      blueprint.size(),
		blueprint: blueprint,
	}
}

func CreateForString[T ~string](maxLength uint32) (Serializer[T], error) {
	blueprint, err := handleType(newContext(), reflect.TypeFor[T](), config{length: maxLength})
	if err != nil {
		return Serializer[T]{}, err
	}
	return Serializer[T]{
		size:      blueprint.size(),
		blueprint: blueprint,
	}, err
}

func CreateForStringSlice[T interface{ ~[]E }, E interface{ ~string | ~*string }](
	maxLength uint32, maxStringLength uint32,
) (Serializer[T], error) {
	config := config{length: maxLength, elementLength: maxStringLength}
	blueprint, err := handleType(newContext(), reflect.TypeFor[T](), config)
	if err != nil {
		return Serializer[T]{}, err
	}
	return Serializer[T]{
		size:      blueprint.size(),
		blueprint: blueprint,
	}, err
}

func CreateForSlice[T interface{ ~[]E }, E any](maxLength uint32) (Serializer[T], error) {
	blueprint, err := handleType(newContext(), reflect.TypeFor[T](), config{length: maxLength})
	if err != nil {
		return Serializer[T]{}, err
	}
	return Serializer[T]{
		size:      blueprint.size(),
		blueprint: blueprint,
	}, err
}

func Create[T any]() (Serializer[T], error) {
	blueprint, err := handleType(newContext(), reflect.TypeFor[T](), config{})
	if err != nil {
		return Serializer[T]{}, err
	}
	return Serializer[T]{
		size:      blueprint.size(),
		blueprint: blueprint,
	}, nil
}

func CreateSliceForSerializer[T any](serializer Serializer[T], maxLength uint32) (Serializer[[]T], error) {
	if maxLength == 0 {
		return Serializer[[]T]{}, ErrLengthMustBeSet
	}
	return Serializer[[]T]{
		blueprint: sliceBlueprint{
			length:  maxLength,
			element: serializer.blueprint,
		},
		size: uint(maxLength) * serializer.size,
	}, nil
}

type Tuple[F, S any] struct {
	First  F
	Second S
}

func CreateForTuple[F, S any](first Serializer[F], second Serializer[S]) Serializer[Tuple[F, S]] {
	return Serializer[Tuple[F, S]]{
		blueprint: tupleBlueprint{
			first:  first.blueprint,
			second: second.blueprint,
		},
		size: first.size + second.size,
	}
}

type config struct {
	length, elementLength uint32
	ignore                bool // only used for struct fields
}

func newContext() context {
	return context{
		seenStructTypes: make(map[reflect.Type]struct{}),
	}
}

const (
	separator      = ";"
	valueSeparator = "="
	tagName        = "eternal"

	elementSizeTag = "elementsize"
	sizeTag        = "size"
	ignoredTag     = "ignored"
)

func parseConfig(raw string) (config, error) {
	var config config
	for _, property := range strings.Split(raw, separator) {
		var split = strings.SplitN(property, valueSeparator, 2)
		switch property := strings.ToLower(strings.TrimSpace(split[0])); property {
		case sizeTag:
			if len(split) == 1 {
				return config, fmt.Errorf("%w: property size must have value set", ErrInvalidAnnotation)
			}
			sizeString := strings.TrimSpace(split[1])
			size, err := strconv.ParseUint(sizeString, 10, 32)
			if err != nil {
				return config, fmt.Errorf("%w: property size be of type uint: %w", ErrInvalidAnnotation, err)
			}
			config.length = uint32(size)
		case elementSizeTag:
			if len(split) == 1 {
				return config, fmt.Errorf("%w: property elementSize must have value set", ErrInvalidAnnotation)
			}
			sizeString := strings.TrimSpace(split[1])
			size, err := strconv.ParseUint(sizeString, 10, 32)
			if err != nil {
				return config, fmt.Errorf("%w: property elementSize be of type uint: %w", ErrInvalidAnnotation, err)
			}
			config.elementLength = uint32(size)
		case ignoredTag:
			config.ignore = true
		case "":
			continue
		default:
			return config, fmt.Errorf("%w: unkown property %s", ErrInvalidAnnotation, property)
		}
	}

	return config, nil
}

type context struct {
	seenStructTypes map[reflect.Type]struct{} // prevents recursive definitions
}

var parsedStructs sync.Map

func handleType(ctx context, _type reflect.Type, blueprintConfig config) (blueprint, error) {
	switch _type.Kind() {
	case reflect.Bool:
		blueprint := boolBlueprint{}
		return blueprint, nil
	case reflect.Int:
		var blueprint blueprint
		switch bits.UintSize {
		case 32:
			blueprint = int32Blueprint{}
		case 64:
			blueprint = int64Blueprint{}
		default:
			panic("int must be 32 or 64 bits")
		}
		return blueprint, nil
	case reflect.Int8:
		blueprint := int8Blueprint{}
		return blueprint, nil
	case reflect.Int16:
		blueprint := int16Blueprint{}
		return blueprint, nil
	case reflect.Int32:
		blueprint := int32Blueprint{}
		return blueprint, nil
	case reflect.Int64:
		blueprint := int64Blueprint{}
		return blueprint, nil
	case reflect.Uint:
		var blueprint blueprint
		switch bits.UintSize {
		case 32:
			blueprint = uint32Blueprint{}
		case 64:
			blueprint = uint64Blueprint{}
		default:
			panic("int must be 32 or 64 bits")
		}
		return blueprint, nil
	case reflect.Uint8:
		blueprint := uint8Blueprint{}
		return blueprint, nil
	case reflect.Uint16:
		blueprint := uint16Blueprint{}
		return blueprint, nil
	case reflect.Uint32:
		blueprint := uint32Blueprint{}
		return blueprint, nil
	case reflect.Uint64:
		blueprint := uint64Blueprint{}
		return blueprint, nil
	case reflect.Float32:
		blueprint := float32Blueprint{}
		return blueprint, nil
	case reflect.Float64:
		blueprint := float64Blueprint{}
		return blueprint, nil
	case reflect.Complex64:
		blueprint := complex64Blueprint{}
		return blueprint, nil
	case reflect.Complex128:
		blueprint := complex128Blueprint{}
		return blueprint, nil
	case reflect.Pointer:
		inner, err := handleType(ctx, _type.Elem(), blueprintConfig)
		if err != nil {
			return nil, err
		}
		return pointerBlueprint{
			element:   inner,
			childSize: inner.size(),
		}, nil
	case reflect.Array:
		inner, err := handleType(ctx, _type.Elem(), config{
			length: blueprintConfig.elementLength,
		})
		return arrayBlueprint{
			element: inner,
			length:  uint(_type.Len()), // overflow cannot happen as uint has more positive values than int
		}, err
	case reflect.Slice:
		if blueprintConfig.length != 0 {
			inner, err := handleType(ctx, _type.Elem(), config{
				length: blueprintConfig.elementLength,
			})
			blueprint := sliceBlueprint{
				element: inner,
				length:  blueprintConfig.length,
			}

			return blueprint, err
		}
		return nil, ErrLengthMustBeSet
	case reflect.String:
		if blueprintConfig.length != 0 {
			blueprint := stringBlueprint{length: blueprintConfig.length}
			return blueprint, nil
		}
		return nil, ErrLengthMustBeSet
	case reflect.Struct:
		if _, ok := ctx.seenStructTypes[_type]; ok {
			return nil, fmt.Errorf("could not handle type %s: %w", _type, ErrRecursiveStructDefinition)
		}
		if blueprint, found := parsedStructs.Load(_type); found {
			typed := blueprint.(structBlueprint)
			return typed, nil
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
				return nil, fmt.Errorf("could not parse config for property %s of type %s: %w", field.Name, _type,
					err)
			}
			if config.ignore {
				continue
			}
			fieldBlueprint, err := handleType(ctx, field.Type, config)
			if err != nil {
				return nil, fmt.Errorf("could not parse config for property %s of type %s: %w", field.Name, _type,
					err)
			}
			size += fieldBlueprint.size()
			blueprint.fields = append(blueprint.fields, structField{
				blueprint:  fieldBlueprint,
				fieldIndex: i,
			})
		}
		blueprint.fields = slices.Clip(blueprint.fields)
		blueprint.totalSize = size
		parsedStructs.Store(_type, blueprint)
		return blueprint, nil
	default:
		return nil, fmt.Errorf("type %s: %w", _type.String(), ErrUnsupportedType)
	}
}
