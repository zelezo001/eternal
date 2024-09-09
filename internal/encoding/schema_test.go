package encoding

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pointerBlueprint(t *testing.T) {
	t.Parallel()
	var inner boolBlueprint
	blueprint := pointerBlueprint{
		childSize: inner.size(),
		element:   inner,
	}
	type Scenario struct {
		Value *bool
		Bytes [2]byte
	}
	scenarios := []Scenario{
		{
			Value: nil,
			Bytes: [2]byte{0, 0},
		},
		{
			Value: pointer(false),
			Bytes: [2]byte{1, 0},
		},
		{
			Value: pointer(true),
			Bytes: [2]byte{1, 1},
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			t.Run("from", func(t *testing.T) {
				var dest *bool
				blueprint.from(scenario.Bytes[:], reflect.ValueOf(&dest).Elem())
				assert.Equal(t, scenario.Value, dest)
			})
			t.Run("to", func(t *testing.T) {
				var dest [2]byte
				blueprint.to(reflect.ValueOf(scenario.Value), dest[:])
				assert.Equal(t, scenario.Bytes[:], dest[:])
			})
		})
	}
}

func Test_tupleBlueprint(t *testing.T) {
	t.Parallel()
	blueprint := tupleBlueprint{
		first:  boolBlueprint{},
		second: uint8Blueprint{},
	}
	type Scenario struct {
		Value Tuple[bool, uint8]
		Bytes [2]byte
	}
	scenarios := []Scenario{
		{
			Value: Tuple[bool, uint8]{First: false, Second: 20},
			Bytes: [2]byte{0, 20},
		},
		{
			Value: Tuple[bool, uint8]{First: true, Second: 0},
			Bytes: [2]byte{1, 0},
		},
		{
			Value: Tuple[bool, uint8]{First: false, Second: 255},
			Bytes: [2]byte{0, 255},
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			t.Run("from", func(t *testing.T) {
				var dest Tuple[bool, uint8]
				blueprint.from(scenario.Bytes[:], reflect.ValueOf(&dest).Elem())
				assert.Equal(t, scenario.Value, dest)
			})
			t.Run("to", func(t *testing.T) {
				var dest [2]byte
				blueprint.to(reflect.ValueOf(scenario.Value), dest[:])
				assert.Equal(t, scenario.Bytes[:], dest[:])
			})
		})
	}
}

func Test_arrayBlueprint(t *testing.T) {
	t.Parallel()
	blueprint := arrayBlueprint{
		length:  5,
		element: uint8Blueprint{},
	}
	type Scenario struct {
		Value [5]uint8
		Bytes [5]byte
	}
	scenarios := []Scenario{
		{
			Value: [5]uint8{1, 2, 3, 4, 5},
			Bytes: [5]byte{1, 2, 3, 4, 5},
		},
		{
			Value: [5]uint8{255, 255, 0, 255, 255},
			Bytes: [5]byte{255, 255, 0, 255, 255},
		},
		{
			Value: [5]uint8{11, 11, 0, 255, 255},
			Bytes: [5]byte{11, 11, 0, 255, 255},
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			t.Run("from", func(t *testing.T) {
				var dest [5]uint8
				blueprint.from(scenario.Bytes[:], reflect.ValueOf(&dest).Elem())
				assert.Equal(t, scenario.Value, dest)
			})
			t.Run("to", func(t *testing.T) {
				var dest [5]byte
				blueprint.to(reflect.ValueOf(scenario.Value), dest[:])
				assert.Equal(t, scenario.Bytes[:], dest[:])
			})
		})
	}
}

func Test_stringBlueprint(t *testing.T) {
	t.Parallel()
	blueprint := stringBlueprint{
		length: 5,
	}
	type Scenario struct {
		Value          string
		ValueFromBytes string
		Bytes          [9]byte
	}
	scenarios := []Scenario{
		{
			Value:          "abcde",
			ValueFromBytes: "abcde",
			Bytes:          [9]byte{0, 0, 0, 5, 'a', 'b', 'c', 'd', 'e'},
		},
		{
			Value:          "try",
			ValueFromBytes: "try",
			Bytes:          [9]byte{0, 0, 0, 3, 't', 'r', 'y'},
		},
		{
			// trimming with respect to encoding
			Value:          "🌈č 4 bytes",
			ValueFromBytes: "🌈",
			Bytes:          [9]byte{0, 0, 0, 4, 0xf0, 0x9f, 0x8c, 0x88},
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			t.Run("from", func(t *testing.T) {
				var dest string
				blueprint.from(scenario.Bytes[:], reflect.ValueOf(&dest).Elem())
				assert.Equal(t, scenario.ValueFromBytes, dest)
			})
			t.Run("to", func(t *testing.T) {
				var dest [9]byte
				blueprint.to(reflect.ValueOf(scenario.Value), dest[:])
				assert.Equal(t, scenario.Bytes[:], dest[:])
			})
		})
	}
}

func Test_sliceBlueprint(t *testing.T) {
	t.Parallel()
	blueprint := sliceBlueprint{
		element: uint8Blueprint{},
		length:  4,
	}
	type Scenario struct {
		Value          []uint8
		ValueFromBytes []uint8
		Bytes          [8]byte
	}
	scenarios := []Scenario{
		{
			// only 4 values will be stored
			Value:          []uint8{4, 5, 6, 10, 22},
			ValueFromBytes: []uint8{4, 5, 6, 10},
			Bytes:          [8]byte{0, 0, 0, 4, 4, 5, 6, 10},
		},
		{
			Value:          []uint8{1, 2},
			ValueFromBytes: []uint8{1, 2},
			Bytes:          [8]byte{0, 0, 0, 2, 1, 2},
		},
		{
			Value:          nil,
			ValueFromBytes: nil,
			Bytes:          [8]byte{},
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			t.Run("from", func(t *testing.T) {
				var dest []uint8
				blueprint.from(scenario.Bytes[:], reflect.ValueOf(&dest).Elem())
				assert.Equal(t, scenario.ValueFromBytes, dest)
			})
			t.Run("to", func(t *testing.T) {
				var dest [8]byte
				blueprint.to(reflect.ValueOf(scenario.Value), dest[:])
				assert.Equal(t, scenario.Bytes[:], dest[:])
			})
		})
	}
}

func Test_struct(t *testing.T) {
	t.Parallel()
	type Persisted struct {
		Active      bool
		Ignored     string
		Value       int16
		Description string
	}
	_type := reflect.TypeFor[Persisted]()
	blueprint := structBlueprint{
		structType: _type,
		fields: []structField{
			{
				blueprint:  boolBlueprint{},
				fieldIndex: 0,
			},
			{
				blueprint:  int16Blueprint{},
				fieldIndex: 2,
			},
			{
				blueprint: stringBlueprint{
					length: 2,
				},
				fieldIndex: 3,
			},
		},
		totalSize: 9,
	}
	type Scenario struct {
		Value          Persisted
		ValueFromBytes Persisted
		Bytes          [9]byte
	}
	scenarios := []Scenario{
		{
			Value: Persisted{
				Active: true,
				Value:  -255,
			},
			ValueFromBytes: Persisted{
				Active: true,
				Value:  -255,
			},
			Bytes: [9]byte{
				1, 255, 1,
			},
		},
		{
			Value: Persisted{
				Active:      false,
				Ignored:     "IGNORED_VALUE",
				Value:       16,
				Description: "abc",
			},
			ValueFromBytes: Persisted{
				Active:      false,
				Value:       16,
				Description: "ab",
			},
			Bytes: [9]byte{
				0, 0, 16, 0, 0, 0, 2, 'a', 'b',
			},
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			t.Run("from", func(t *testing.T) {
				var dest Persisted
				blueprint.from(scenario.Bytes[:], reflect.ValueOf(&dest).Elem())
				assert.Equal(t, scenario.ValueFromBytes, dest)
			})
			t.Run("to", func(t *testing.T) {
				var dest [9]byte
				blueprint.to(reflect.ValueOf(scenario.Value), dest[:])
				assert.Equal(t, scenario.Bytes[:], dest[:])
			})
		})
	}
}

func pointer[T any](value T) *T {
	return &value
}
