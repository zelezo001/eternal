package eternal

import (
	"cmp"
	"errors"
	"slices"

	"github.com/zelezo001/eternal/internal/stack"
)

type NodeStorage[K cmp.Ordered, V any] interface {
	GetRoot() (Node[K, V], error)
	GetDepth() (uint, error)
	SetDepth(uint) error
	Get(id uint) (Node[K, V], error)
	Persist(node Node[K, V]) error
	Remove(id uint) error
	NewId() (uint, error)
}

type tree[K cmp.Ordered, V any] struct {
	a, b    uint
	depth   uint
	storage NodeStorage[K, V]
}

var ErrNotFound = errors.New("value not found")

func persistMultiple[K cmp.Ordered, V any](storage NodeStorage[K, V], nodes ...Node[K, V]) error {
	for _, n := range nodes {
		if err := storage.Persist(n); err != nil {
			return err
		}
	}

	return nil
}

func (t tree[K, V]) Get(key K) (V, error) {
	var emptyValue V
	root, err := t.storage.GetRoot()
	if err != nil {
		return emptyValue, err
	}
	var currentNode = root
	for {
		found, position, pair := currentNode.values.find(key)
		if found {
			return pair.second, nil
		}
		if currentNode.leaf {
			return emptyValue, ErrNotFound
		}
		// presence of position is guarantied by nature of (a,b)-tree
		var nextNodeId = currentNode.children[position]
		currentNode, err = t.storage.Get(nextNodeId)
		if err != nil {
			return emptyValue, err
		}
	}
}
func (t *tree[K, V]) Insert(key K, value V) error {
	path := stack.NewStack[uint](t.depth)
	root, err := t.storage.GetRoot()
	if err != nil {
		return err
	}
	var currentNode = root
	for {
		path.Push(currentNode.id)
		found, position, _ := currentNode.values.find(key)
		if found {
			currentNode.values[position] = tuple[K, V]{key, value}
			return t.storage.Persist(currentNode)
		}
		if currentNode.leaf {
			// we cannot persist currentNode as it can violate a-b rules
			currentNode.values.add(tuple[K, V]{key, value})
			break
		}
		// presence of position is guarantied by nature of (a,b)-tree
		var nextNodeId = currentNode.children[position]
		currentNode, err = t.storage.Get(nextNodeId)
		if err != nil {
			return err
		}
	}

	for {
		if currentNode.values.count() < t.b {
			return nil
		}
		if path.Empty() {
			newNodeId, err := t.storage.NewId()
			if err != nil {
				return err
			}
			newNode, middle, oldNode := t.splitNode(newNodeId, currentNode)
			replacedNodeId, err := t.storage.NewId()
			if err != nil {
				return err
			}
			newRoot := createNewNode[K, V](t.b, oldNode.id, false)
			oldNode.id = replacedNodeId

			newRoot.values.add(middle)
			newRoot.children = append(currentNode.children, newNode.id, oldNode.id)

			if err := persistMultiple(t.storage, currentNode, oldNode, newNode); err != nil {
				return err
			}
			t.depth++
			return t.storage.SetDepth(t.depth)
		} else {
			parent, err := t.storage.Get(path.Pop())
			if err != nil {
				return err
			}
			newNodeId, err := t.storage.NewId()
			if err != nil {
				return err
			}

			newNode, middle, oldNode := t.splitNode(newNodeId, currentNode)
			parent.children = append(parent.children, 0)
			var toReplace = len(parent.children)
			for toReplace > 0 {
				var copied = parent.children[toReplace-1]
				parent.children[toReplace] = copied
				if copied == oldNode.id {
					toReplace--
					parent.children[toReplace] = newNode.id
				}
				copied--
			}
			parent.values.add(middle)
			if err := persistMultiple(t.storage, parent, oldNode, newNode); err != nil {
				return err
			}

			currentNode = parent
		}
	}

}

func (t *tree[K, V]) splitNode(newNodeId uint, currentNode Node[K, V]) (Node[K, V], tuple[K, V], Node[K, V]) {
	newNode := Node[K, V]{
		id:       newNodeId,
		values:   make([]tuple[K, V], 0, t.b),
		children: make([]uint, 0, t.b+1),
		leaf:     currentNode.leaf,
	}
	middleIndex := (t.b) / 2
	middle := currentNode.values[middleIndex]
	for i := uint(0); i < middleIndex; i++ {
		newNode.values = append(newNode.values, currentNode.values[i])
		newNode.children = append(newNode.children, currentNode.children[i])

		currentNode.values[i] = currentNode.values[1+i+middleIndex]
		currentNode.children[i] = currentNode.children[1+i+middleIndex]
	}
	newNode.children = append(newNode.children, currentNode.children[middleIndex])
	currentNode.children[middleIndex] = currentNode.children[t.b]

	return newNode, middle, currentNode
}

type Node[K cmp.Ordered, V any] struct {
	id       uint
	values   values[K, V]
	children []uint
	leaf     bool
}

type values[K cmp.Ordered, V any] []tuple[K, V]

func (values values[K, V]) count() uint {
	return uint(len(values))
}

func (values values[K, V]) find(key K) (bool, int, tuple[K, V]) {
	position, found := slices.BinarySearchFunc(values, key, func(t tuple[K, V], k K) int {
		return cmp.Compare(t.first, k)
	})
	if found {
		return true, position, values[position]
	}

	return false, position, tuple[K, V]{}
}

func (find *values[K, V]) add(value tuple[K, V]) {
	*find = append(*find, value)
	slices.SortFunc(*find, func(a, b tuple[K, V]) int {
		return cmp.Compare(a.first, b.first)
	})
}

func createNewNode[K cmp.Ordered, V any](b, id uint, leaf bool) Node[K, V] {
	return Node[K, V]{
		id:       id,
		values:   make([]tuple[K, V], 0, b),
		children: make([]uint, 0, b+1),
		leaf:     leaf,
	}
}

type tuple[A, B any] struct {
	first  A
	second B
}
