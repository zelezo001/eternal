package encoding

import (
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_handleType(t *testing.T) {
	t.Parallel()
	type String string
	type Composed struct {
		Name      *string `eternal:"size=10"`
		IntValues [3]int32
		Ignored   io.Reader `eternal:"ignored"`
		Embedded  struct {
			Name   string   `eternal:"size=11"`
			Values []string `eternal:"size=2;elementSize=4"`
		}
	}
	type Scenario struct {
		Blueprint    blueprint
		ExpectedSize uint
		Config       config
		Type         reflect.Type
	}
	scenarios := []Scenario{
		{
			Blueprint: structBlueprint{
				structType: reflect.TypeFor[Composed](),
				fields: []structField{
					{
						blueprint: pointerBlueprint{
							childSize: 14,
							element:   stringBlueprint{10},
						},
						fieldIndex: 0,
					},
					{
						blueprint: arrayBlueprint{
							length:  3,
							element: int32Blueprint{},
						},
						fieldIndex: 1,
					},
					{
						blueprint: structBlueprint{
							structType: reflect.TypeFor[struct {
								Name   string   `eternal:"size=11"`
								Values []string `eternal:"size=2;elementSize=4"`
							}](),
							fields: []structField{
								{
									blueprint:  stringBlueprint{length: 11},
									fieldIndex: 0,
								},
								{
									blueprint: sliceBlueprint{
										length: 2,
										element: stringBlueprint{
											length: 4,
										},
									},
									fieldIndex: 1,
								},
							},
							totalSize: 35,
						},
						fieldIndex: 3,
					},
				},
				totalSize: 62,
			},
			ExpectedSize: 62,
			Config:       config{},
			Type:         reflect.TypeFor[Composed](),
		},
		{
			Blueprint:    stringBlueprint{length: 50},
			ExpectedSize: 54,
			Config: config{
				length:        50,
				elementLength: 0,
				ignore:        false,
			},
			Type: reflect.TypeFor[string](),
		},
		{
			Blueprint:    stringBlueprint{length: 50},
			ExpectedSize: 54,
			Config: config{
				length:        50,
				elementLength: 0,
				ignore:        false,
			},
			Type: reflect.TypeFor[String](),
		},
		{
			Blueprint: sliceBlueprint{
				length:  2,
				element: stringBlueprint{length: 50},
			},
			ExpectedSize: 54*2 + 4,
			Config: config{
				length:        2,
				elementLength: 50,
				ignore:        false,
			},
			Type: reflect.TypeFor[[]string](),
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			blueprint, err := handleType(newContext(), scenario.Type, scenario.Config)
			assert.NoError(t, err)
			assert.Equal(t, scenario.ExpectedSize, blueprint.size())
			assert.Equal(t, scenario.Blueprint, blueprint)
		})
	}
}
