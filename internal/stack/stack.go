package stack

type Stack[T any] struct {
	values []T
}

func NewStack[T any](preallocate uint) *Stack[T] {
	return &Stack[T]{
		values: make([]T, 0, preallocate),
	}
}

func (receiver *Stack[T]) Pop() T {
	index := len(receiver.values) - 1
	value := receiver.values[index]
	receiver.values = receiver.values[:index]
	return value
}

func (receiver *Stack[T]) Push(value T) {
	receiver.values = append(receiver.values, value)
}

func (receiver *Stack[T]) Count() uint {
	return uint(len(receiver.values))
}

func (receiver *Stack[T]) Empty() bool {
	return len(receiver.values) == 0
}
