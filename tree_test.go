package eternal

import (
	"cmp"
	"slices"
	"testing"

	"github.com/zelezo001/eternal/encoding"
)

func createTreeWithInMemoryStorage[K cmp.Ordered, T any](a, b uint) (*Tree[K, T], *InMemoryStorage[K, T]) {
	storage := InMemory[K, T](b)
	tree, err := NewTree[K, T](a, b, storage)
	if err != nil {
		panic(err)
	}

	return tree, storage
}

func TestTree(t *testing.T) {
	t.Parallel()
	values := []encoding.Tuple[string, uint]{
		{
			First:  "KEY_1",
			Second: 1,
		},
		{
			First:  "KEY_2",
			Second: 2,
		},
		{
			First:  "KEY_3",
			Second: 3,
		},
		{
			First:  "KEY_4",
			Second: 3,
		},
		{
			First:  "KEY_3",
			Second: 5,
		},
		{
			First:  "KEY_5",
			Second: 5,
		},
		{
			First:  "KEY_6",
			Second: 5,
		},
		{
			First:  "KEY_7",
			Second: 5,
		},
	}
	flattened := make(map[string]uint, len(values))
	tree, storage := createTreeWithInMemoryStorage[string, uint](2, 3)
	for _, value := range values {
		flattened[value.First] = value.Second
		err := tree.Insert(value.First, value.Second)
		if err != nil {
			t.Fatalf("failed storing value: %s", err)
		}
	}
	for key, expectedValue := range flattened {
		value, err := tree.Get(key)
		if err != nil {
			t.Fatalf("failed getting value with key %s: %s", key, err)
		}
		if value != expectedValue {
			t.Fatalf("exected value %d for key %s, got %d", expectedValue, key, value)
		}
	}
	checker := &treeChecker[string, uint]{
		testing:           t,
		storage:           storage,
		expectedDepth:     3,
		a:                 2,
		b:                 3,
		checkedNodes:      make(map[uint]struct{}, 7),
		expectedNodeCount: 7,
	}
	checker.checkTree()

	const deletedKey = "KEY_4"
	err := tree.Delete(deletedKey)
	if err != nil {
		t.Fatalf("failed deleting value with key %s: %s", deletedKey, err)
	}

	checker = &treeChecker[string, uint]{
		testing:           t,
		storage:           storage,
		expectedDepth:     2,
		a:                 2,
		b:                 3,
		checkedNodes:      make(map[uint]struct{}, 4),
		expectedNodeCount: 4,
	}
	checker.checkTree()
}

func TestTree_3_5(t *testing.T) {
	t.Parallel()
	const a, b uint = 3, 5
	data := []int64{
		6148, 7815, 4233, 3537, 9251, 4264, 5908, 4758, 4392, 3129, 8362, 4937, 778, 7740, 4774, 1227, 9441, 7328, 6167,
		3641, 6796, 9364, 2491, 7683, 4164, 7985, 4609, 1034, 878, 585, 4009, 1517, 1446, 6831, 6900, 3609, 1548, 896,
		2362, 7365, 6166, 9420, 8999, 4704, 5238, 3296, 4646, 508, 1355, 937, 3315, 2144, 2134, 8504, 4459, 9907, 4258,
		9952, 2552, 5598, 6808, 1830, 1518, 3379, 7818, 5495, 8920, 6508, 5530, 9362, 8498, 7447, 1851, 5641, 911, 9810,
		6595, 4989, 8071, 4234, 8688, 1095, 8742, 1433, 3296, 2314, 3587, 439, 9979, 5751, 1300, 8698, 8948, 2027, 6098,
		2117, 1931, 7393, 8097, 2015,
	}
	toDelete := []int64{
		440, 6540, 8957, 2027, 3315, 4646, 4234, 9251, 2420, 9480, 6595, 8698, 1517, 7631, 5495, 3953, 5012, 2314, 7885,
		2800,
	}

	tree, storage := createTreeWithInMemoryStorage[int64, int64](a, b)
	for _, value := range data {
		err := tree.Insert(value, value)
		if err != nil {
			t.Fatalf("failed inserting value: %s", err)
		}
	}
	(&treeChecker[int64, int64]{
		testing:       t,
		storage:       storage,
		expectedDepth: 0,
		a:             a,
		b:             b,
		checkedNodes:  make(map[uint]struct{}),
	}).checkTree()

	for _, value := range toDelete {
		err := tree.Delete(value)
		if err != nil {
			t.Fatalf("failed inserting value: %s", err)
		}
	}

	(&treeChecker[int64, int64]{
		testing:       t,
		storage:       storage,
		expectedDepth: 0,
		a:             a,
		b:             b,
		checkedNodes:  make(map[uint]struct{}),
	}).checkTree()
}

type treeChecker[K cmp.Ordered, V any] struct {
	testing           *testing.T
	storage           NodeStorage[K, V]
	expectedDepth     uint // set to zero for avoiding global depth test
	a, b              uint
	checkedNodes      map[uint]struct{}
	expectedNodeCount int
}

func (t *treeChecker[K, V]) checkTree() {
	t.testing.Helper()
	if t.expectedDepth == 0 {
		t.expectedDepth = t.storage.GetDepth()
	}
	if t.expectedDepth != t.storage.GetDepth() {
		t.testing.Fatalf("unexpected depth %d, expected %d", t.storage.GetDepth(), t.expectedDepth)
	}
	root, err := t.storage.GetRoot()
	if err != nil {
		t.testing.Fatalf("could not fetch root: %s", err)
	}
	t.checkNode(root.id, 1, nil, nil)
	if t.expectedNodeCount != 0 && len(t.checkedNodes) != t.expectedNodeCount {
		t.testing.Fatalf("expected exactly %d nodes, %d nodes found", t.expectedNodeCount, len(t.checkedNodes))
	}
}

func (t *treeChecker[K, V]) checkNode(nodeId, depth uint, min, max *K) {
	t.testing.Helper()
	if _, ok := t.checkedNodes[nodeId]; ok {
		t.testing.Fatalf("node %d visited twice", nodeId)
	}
	t.checkedNodes[nodeId] = struct{}{}
	node, err := t.storage.Get(nodeId)
	if err != nil {
		t.testing.Fatalf("could not fetch node %d due to: %s", nodeId, err)
	}
	if node.leaf && len(node.children) != 0 {
		t.testing.Fatalf("node %d is leaf with children", nodeId)
	}
	if node.leaf && t.expectedDepth != depth {
		t.testing.Fatalf("node %d is leaf but on unexpected depth %d", nodeId, depth)
	}
	if uint(len(node.values)) > t.b-1 {
		t.testing.Fatalf("node %d: number of values is more than b-1", nodeId)
	}
	if uint(len(node.values)) < t.a-1 && node.id != 0 {
		t.testing.Fatalf("node %d: number of values is more less than a-1", nodeId)
	}
	if !node.leaf && len(node.values)+1 != len(node.children) {
		t.testing.Fatalf("node %d: number of children should be one more than number of values", nodeId)
	}
	if !slices.IsSortedFunc(node.values, func(a, b encoding.Tuple[K, V]) int {
		return cmp.Compare(a.First, b.First)
	}) {
		t.testing.Fatalf("node %d: values are not sorted", nodeId)
	}
	if min != nil {
		if slices.ContainsFunc(node.values, func(e encoding.Tuple[K, V]) bool {
			return e.First <= *min
		}) {
			t.testing.Fatalf("node %d does not respect order", nodeId)
		}
	}
	if max != nil {
		if slices.ContainsFunc(node.values, func(e encoding.Tuple[K, V]) bool {
			return e.First >= *max
		}) {
			t.testing.Fatalf("node %d does not respect order", nodeId)
		}
	}
	for i, child := range node.children {
		var childMin, childMax *K
		if i != 0 {
			childMin = &node.values[i-1].First
		} else {
			// leftmost child, minimal value is same as parents
			childMin = min
		}
		if i != len(node.children)-1 {
			childMax = &node.values[i].First
		} else {
			// leftmost child, max value is same as parents
			childMax = max
		}
		t.checkNode(child, depth+1, childMin, childMax)
	}
}
