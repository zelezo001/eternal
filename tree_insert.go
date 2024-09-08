package eternal

import (
	"cmp"

	"github.com/zelezo001/eternal/internal/encoding"
	"github.com/zelezo001/eternal/internal/stack"
)

func (t *Tree[K, V]) Insert(key K, value V) error {
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
			currentNode.values[position] = encoding.Tuple[K, V]{First: key, Second: value}
			return t.storage.Persist(currentNode)
		}
		if currentNode.leaf {
			// we cannot persist currentNode as it can violate a-b rules
			currentNode.values.add(encoding.Tuple[K, V]{First: key, Second: value})
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

func persistMultiple[K cmp.Ordered, V any](storage NodeStorage[K, V], nodes ...Node[K, V]) error {
	for _, n := range nodes {
		if err := storage.Persist(n); err != nil {
			return err
		}
	}

	return nil
}

func (t *Tree[K, V]) splitNode(newNodeId uint, currentNode Node[K, V]) (Node[K, V], encoding.Tuple[K, V], Node[K, V]) {
	newNode := createNewNode[K, V](t.b, newNodeId, currentNode.leaf)
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

func createNewNode[K cmp.Ordered, V any](b, id uint, leaf bool) Node[K, V] {
	return Node[K, V]{
		id:       id,
		values:   make([]encoding.Tuple[K, V], 0, b),
		children: make([]uint, 0, b+1),
		leaf:     leaf,
	}
}
