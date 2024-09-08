package eternal

import (
	"reflect"
	"slices"
	"testing"
)

func Test_pop(t *testing.T) {
	t.Parallel()
	type Scenario struct {
		ExpectedSlice []int
		ExpectedValue int
		Slice         []int
		Position      uint
	}
	scenarios := []Scenario{
		{
			ExpectedSlice: []int{2},
			ExpectedValue: 1,
			Slice:         []int{1, 2},
			Position:      0,
		},
		{
			ExpectedSlice: []int{1, 2, 5},
			ExpectedValue: 6,
			Slice:         []int{1, 2, 5, 6},
			Position:      3,
		},
		{
			ExpectedSlice: []int{1, 2, 6},
			ExpectedValue: 5,
			Slice:         []int{1, 2, 5, 6},
			Position:      2,
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			value, slice := pop(scenario.Slice, scenario.Position)
			if value != scenario.ExpectedValue || slices.Compare(slice, scenario.ExpectedSlice) != 0 {
				t.Fatalf("tried popping position %d, expected final array to be %v and value %d, but got %v and %d",
					scenario.Position, scenario.ExpectedSlice, scenario.ExpectedValue, slice, value)
			}
		})
	}
}

func Test_prepend(t *testing.T) {
	t.Parallel()
	expected := []int{55, 1, 2, 3}
	input := []int{1, 2, 3}
	value := 55

	slice := prepend(input, value)
	if slices.Compare(slice, expected) != 0 {
		t.Fatalf("expected %v, got %v", expected, slice)
	}
}

func Test_prependBefore(t *testing.T) {
	t.Parallel()
	type Scenario struct {
		ExpectedSlice []int
		Slice         []int
		Target        int
		Value         int
	}
	scenarios := []Scenario{
		{
			ExpectedSlice: []int{2, 3, 4, 10, 11, 23, 10},
			Slice:         []int{2, 3, 4, 10, 11, 10},
			Target:        10,
			Value:         23,
		},
		{
			ExpectedSlice: []int{1, 2, 5, 6},
			Slice:         []int{1, 2, 6},
			Target:        6,
			Value:         5,
		},
		{
			ExpectedSlice: []int{-1, 1, 2, 6},
			Slice:         []int{1, 2, 6},
			Target:        1,
			Value:         -1,
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			slice := prependBefore(scenario.Slice, scenario.Value, scenario.Target)
			if slices.Compare(slice, scenario.ExpectedSlice) != 0 {
				t.Fatalf("tried prepent %d before %d, expected final array to be %v, but got %v",
					scenario.Value, scenario.Target, scenario.ExpectedSlice, slice)
			}
		})
	}
}

func Test_swap(t *testing.T) {
	t.Parallel()
	type Scenario struct {
		ExpectedSlice []int
		ExpectedValue int
		Slice         []int
		Value         int
		Position      uint
	}
	scenarios := []Scenario{
		{
			ExpectedSlice: []int{0, 2},
			ExpectedValue: 1,
			Slice:         []int{1, 2},
			Value:         0,
			Position:      0,
		},
		{
			ExpectedSlice: []int{1, 2, 2, 6},
			ExpectedValue: 5,
			Slice:         []int{1, 2, 5, 6},
			Value:         2,
			Position:      2,
		},
	}
	for _, scenario := range scenarios {
		scenario := scenario
		t.Run("", func(t *testing.T) {
			result := swap(scenario.Slice, scenario.Position, scenario.Value)
			if result != scenario.ExpectedValue || !reflect.DeepEqual(scenario.ExpectedSlice, scenario.Slice) {
				t.Fatalf(
					"tried swapping position %d with %d, expected final slice to be %v and value %d to be returned, got %v and %d",
					scenario.Position, scenario.Value, scenario.ExpectedSlice, scenario.ExpectedValue, scenario.Slice,
					result,
				)
			}
		})
	}
}
