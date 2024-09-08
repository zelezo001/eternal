package eternal

import (
	"cmp"
	"errors"
	"slices"

	"github.com/zelezo001/eternal/internal/encoding"
)

type NodeStorage[K cmp.Ordered, V any] interface {
	GetRoot() (Node[K, V], error)
	GetDepth() uint
	SetDepth(uint) error
	Get(id uint) (Node[K, V], error)
	Persist(node Node[K, V]) error
	Remove(id uint) error
	NewId() (uint, error)
}

type Tree[K cmp.Ordered, V any] struct {
	a, b    uint
	depth   uint
	storage NodeStorage[K, V]
}

func NewTree[K cmp.Ordered, V any](a, b uint, storage NodeStorage[K, V]) *Tree[K, V] {
	return &Tree[K, V]{
		a:       a,
		b:       b,
		depth:   storage.GetDepth(),
		storage: storage,
	}
}

var ErrNotFound = errors.New("value not found")

func (t *Tree[K, V]) Get(key K) (V, error) {
	var emptyValue V
	root, err := t.storage.GetRoot()
	if err != nil {
		return emptyValue, err
	}
	var currentNode = root
	for {
		found, position, pair := currentNode.values.find(key)
		if found {
			return pair.Second, nil
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

type Node[K cmp.Ordered, V any] struct {
	id       uint
	values   values[K, V]
	children []uint
	leaf     bool
}

type values[K cmp.Ordered, V any] []encoding.Tuple[K, V]

func (values *values[K, V]) count() uint {
	return uint(len(*values))
}

func (values *values[K, V]) find(key K) (bool, int, encoding.Tuple[K, V]) {
	position, found := slices.BinarySearchFunc(*values, key, func(t encoding.Tuple[K, V], k K) int {
		return cmp.Compare(t.First, k)
	})
	if found {
		return true, position, (*values)[position]
	}

	return false, position, encoding.Tuple[K, V]{}
}

func (values *values[K, V]) add(value encoding.Tuple[K, V]) {
	*values = append(*values, value)
	slices.SortFunc(*values, func(a, b encoding.Tuple[K, V]) int {
		return cmp.Compare(a.First, b.First)
	})
}
