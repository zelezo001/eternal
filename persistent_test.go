package eternal

import (
	"math/bits"
	"os"
	"testing"

	"github.com/zelezo001/eternal/encoding"
)

func TestPersistentStorage(t *testing.T) {
	t.Parallel()
	const a, b = 2, 3
	const blockSize = 16
	temp, err := os.CreateTemp(t.TempDir(), "file")
	if err != nil {
		t.Fatalf("could not create file: %s", err)
	}
	keySerializer, err := encoding.CreateForString[string](5)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := NewPersistentStorage[string, uint](a, b, blockSize, temp, keySerializer,
		encoding.CreateForPrimitive[uint]())
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
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
	tree, err := NewTree[string, uint](a, b, storage)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		flattened[value.First] = value.Second
		err := tree.Insert(value.First, value.Second)
		if err != nil {
			t.Fatalf("failed storing value %s: %s", value.First, err)
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
	//printNodes(storage)

	const deletedKey = "KEY_4"
	err = tree.Delete(deletedKey)
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
	err = storage.Defragment()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	checker = &treeChecker[string, uint]{
		testing:            t,
		storage:            storage,
		expectedDepth:      2,
		a:                  2,
		b:                  3,
		checkedNodes:       make(map[uint]struct{}, 4),
		expectedNodeCount:  4,
		expectedValueCount: 6,
	}
	checker.checkTree()
	stat, err := temp.Stat()
	if err != nil {
		t.Fatalf("could not obtain info about file: %s", err)
	}
	const expectedFileSizeAfterTrimming = 320 + 98 + bits.UintSize/8*2 // 4 nodes with padding + header + two metadata uints
	if stat.Size() != expectedFileSizeAfterTrimming {
		t.Fatalf("expected file to have size %d, file has size %d", expectedFileSizeAfterTrimming, stat.Size())
	}
}
