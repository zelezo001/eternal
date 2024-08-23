package eternal

import (
	"cmp"
	"errors"
)

const rootId = 0

type InMemoryStorage[K cmp.Ordered, V any] struct {
	nodes     map[uint]Node[K, V]
	unusedIds stack[uint]
	idCap     uint
}

func InMemory[K cmp.Ordered, V any](b uint) *InMemoryStorage[K, V] {
	return &InMemoryStorage[K, V]{
		nodes: map[uint]Node[K, V]{
			rootId: createNewNode[K, V](b, rootId, true),
		},
		unusedIds: stack[uint]{},
		idCap:     0,
	}
}

func (i *InMemoryStorage[K, V]) GetRoot() (Node[K, V], error) {
	return i.nodes[rootId], nil
}

func (i *InMemoryStorage[K, V]) Get(id uint) (Node[K, V], error) {
	node, found := i.nodes[id]
	if !found {
		return Node[K, V]{}, errors.New("node not found")
	}
	return node, nil
}

func (i *InMemoryStorage[K, V]) Persist(node Node[K, V]) error {
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
		i.unusedIds.add(id)
	}
	return nil
}

func (i *InMemoryStorage[K, V]) NewId() (uint, error) {
	if len(i.unusedIds.values) == 0 {
		id := i.idCap
		i.idCap++
		return id, nil
	}

	return i.unusedIds.pop(), nil
}
