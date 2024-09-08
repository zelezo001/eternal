package eternal

import (
	"cmp"
	"errors"
	"slices"

	"github.com/zelezo001/eternal/internal/encoding"
)

type NodeStorage[K cmp.Ordered, V any] interface {
	// GetRoot
	// Should return root node of stored tree. GetRoot is expected to return valid Node even if it has not yet been stored.
	// Its
	GetRoot() (Node[K, V], error)
	// GetDepth
	// Should return depth of stored tree. Default value of empty tree should be 1. Depth is set by SetDepth and no
	// assumption should be done about the tree outside of default value.
	GetDepth() uint
	// SetDepth
	// See GetDepth
	SetDepth(uint) error
	// Get
	// Should get node according to its id. Only valid ids are expected to be passed.
	Get(id uint) (Node[K, V], error)
	// Persist
	// Only node root and ids obtained from other nodes/NewId method are expected to be passed.
	Persist(node Node[K, V]) error
	// Remove
	// Should remove node according to its id. Only valid ids are expected to be passed.
	Remove(id uint) error
	// NewId
	// Returns ID which is not used by persisted nodes.
	// Returned ID must not be returned again without being removed with Remove first.
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

// Get
// Returns stored value by given key. If value is not present, ErrNotFound is returned.
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
			// we hit leaf, searched key is not in the tree
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

// find
// Search for key-value tuple by key. If false is returned, int value position indicated position
// of key:  values[position-1].First < key < values[position].
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
