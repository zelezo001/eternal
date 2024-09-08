package eternal

import (
	"cmp"
	"errors"

	"github.com/zelezo001/eternal/internal/stack"
)

type InMemoryStorage[K cmp.Ordered, V any] struct {
	nodes     map[uint]node[K, V]
	unusedIds *stack.Stack[uint]
	idCap     uint
	depth     uint
}

var _ NodeStorage[string, any] = &InMemoryStorage[string, any]{}

func InMemory[K cmp.Ordered, V any](b uint) *InMemoryStorage[K, V] {
	return &InMemoryStorage[K, V]{
		nodes: map[uint]node[K, V]{
			rootId: createNewNode[K, V](b, rootId, true),
		},
		unusedIds: stack.NewStack[uint](0),
		idCap:     0,
	}
}

func (i *InMemoryStorage[K, V]) GetDepth() uint {
	return i.depth
}

func (i *InMemoryStorage[K, V]) SetDepth(depth uint) error {
	i.depth = depth
	return nil
}

func (i *InMemoryStorage[K, V]) GetRoot() (node[K, V], error) {
	return i.nodes[rootId], nil
}

func (i *InMemoryStorage[K, V]) Get(id uint) (node[K, V], error) {
	node, found := i.nodes[id]
	if !found {
		return node[K, V]{}, errors.New("node not found")
	}
	return node, nil
}

func (i *InMemoryStorage[K, V]) Persist(node node[K, V]) error {
	i.nodes[node.id] = node
	return nil
}

func (i *InMemoryStorage[K, V]) Remove(id uint) error {
	if id == 0 {
		return nil
	}
	if _, exists := i.nodes[id]; !exists {
		return nil
	}
	delete(i.nodes, id)
	if id+1 == i.idCap {
		i.idCap--
	} else {
		i.unusedIds.Push(id)
	}
	return nil
}

func (i *InMemoryStorage[K, V]) NewId() (uint, error) {
	if i.unusedIds.Empty() {
		id := i.idCap
		i.idCap++
		return id, nil
	}

	return i.unusedIds.Pop(), nil
}
