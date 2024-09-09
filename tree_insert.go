package eternal

import (
	"cmp"

	"github.com/zelezo001/eternal/encoding"
	"github.com/zelezo001/eternal/internal/stack"
)

func (t *Tree[K, V]) Insert(key K, value V) error {
	// path to leaf is always t.depth nodes and the last node is stored in currentNode
	path := stack.NewStack[uint](t.depth - 1)
	root, err := t.storage.GetRoot()
	if err != nil {
		return err
	}
	var currentNode = root
	for {
		found, position, _ := currentNode.values.find(key)
		if found {
			// we don't change number of values in the tree, no back-tracing is needed
			currentNode.values[position] = encoding.Tuple[K, V]{First: key, Second: value}
			return t.storage.Persist(currentNode)
		}
		if currentNode.leaf {
			// we cannot persist currentNode as it can violate a-b rules
			currentNode.values.add(encoding.Tuple[K, V]{First: key, Second: value})
			break
		}
		path.Push(currentNode.id)
		// presence of position is guarantied by nature of (a,b)-tree
		var nextNodeId = currentNode.children[position]
		currentNode, err = t.storage.Get(nextNodeId)
		if err != nil {
			return err
		}
	}

	for {
		if currentNode.values.count() < t.b {
			// currentNode is either node with new value or parent from previous iteration.
			//In both scenarios, we need to persist it.
			return t.storage.Persist(currentNode)
		}
		if path.Empty() {
			// currentNode is root
			newNodeId, err := t.storage.NewId()
			if err != nil {
				return err
			}
			newNode, middle, oldRoot := t.splitFullNode(newNodeId, currentNode)
			oldRootNewId, err := t.storage.NewId()
			if err != nil {
				return err
			}
			// as by contract rootId is unknown, we will get its id from oldRoot
			newRoot := createNewNode[K, V](t.b, oldRoot.id, false)
			newRoot.values.add(middle)
			newRoot.children = append(newRoot.children, newNode.id, oldRootNewId)
			oldRoot.id = oldRootNewId

			if err := persistMultiple(t.storage, newRoot, oldRoot, newNode); err != nil {
				return err
			}
			return t.updateDepth(t.depth + 1)
		} else {
			parent, err := t.storage.Get(path.Pop())
			if err != nil {
				return err
			}
			newNodeId, err := t.storage.NewId()
			if err != nil {
				return err
			}

			newNode, middle, oldNode := t.splitFullNode(newNodeId, currentNode)
			parent.children = prependBefore(parent.children, newNode.id, oldNode.id)
			parent.values.add(middle)
			if err := persistMultiple(t.storage, oldNode, newNode); err != nil {
				return err
			}

			currentNode = parent
		}
	}

}

func persistMultiple[K cmp.Ordered, V any](storage NodeStorage[K, V], nodes ...Node[K, V]) error {
	for _, n := range nodes {
		if err := storage.Persist(n); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tree[K, V]) splitFullNode(newNodeId uint, currentNode Node[K, V]) (
	Node[K, V], encoding.Tuple[K, V], Node[K, V],
) {
	newNode := createNewNode[K, V](t.b, newNodeId, currentNode.leaf)
	middleIndex := (t.b) / 2
	middle := currentNode.values[middleIndex]
	for i := uint(0); i < middleIndex; i++ {
		newNode.values = append(newNode.values, currentNode.values[i])
		currentNode.values[i] = currentNode.values[1+i+middleIndex]

		if !currentNode.leaf {
			newNode.children = append(newNode.children, currentNode.children[i])
			currentNode.children[i] = currentNode.children[1+i+middleIndex]
		}
	}
	if !currentNode.leaf {
		newNode.children = append(newNode.children, currentNode.children[middleIndex])
		currentNode.children[middleIndex] = currentNode.children[t.b]
		currentNode.children = currentNode.children[:middleIndex+1]
	}
	currentNode.values = currentNode.values[:middleIndex]

	return newNode, middle, currentNode
}

func createNewNode[K cmp.Ordered, V any](b, id uint, leaf bool) Node[K, V] {
	return Node[K, V]{
		id:       id,
		values:   make([]encoding.Tuple[K, V], 0, b),
		children: make([]uint, 0, b+1),
		leaf:     leaf,
	}
}
