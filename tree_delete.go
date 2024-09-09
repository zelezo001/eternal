package eternal

import (
	"github.com/zelezo001/eternal/encoding"
	"github.com/zelezo001/eternal/internal/stack"
)

func (t *Tree[K, V]) Delete(key K) error {
	path := stack.NewStack[deleteStep](t.depth)
	root, err := t.storage.GetRoot()
	if err != nil {
		return err
	}
	currentNode := root
	var positionInParent uint
	for {
		path.Push(deleteStep{currentNode.id, positionInParent})
		found, position, _ := currentNode.values.find(key)
		if found {
			if currentNode.leaf {
				_, currentNode.values = pop(currentNode.values, uint(position))
				if err := t.storage.Persist(currentNode); err != nil {
					return err
				}
				break
			} else {
				// presence of position is guarantied by nature of (a,b)-tree
				leftChildId := currentNode.children[position]
				// value is not stored in leaf, we must find predecessor to replace inner value
				// predecessor definitely exists as largest value in every (sub)tree is always in leaf
				valueToReplace, err := t.popLargest(path, leftChildId, uint(position))
				if err != nil {
					return err
				}
				currentNode.values[position] = valueToReplace
				break
			}
		}
		if currentNode.leaf {
			// key is not present in the tree
			return nil
		}
		positionInParent = uint(position)
		// presence of position is guarantied by nature of (a,b)-tree
		var nextNodeId = currentNode.children[position]
		currentNode, err = t.storage.Get(nextNodeId)
		if err != nil {
			return err
		}
	}

	return t.balanceTreeAfterDelete(path)
}

func (t *Tree[K, V]) popLargest(
	visitedNodes *stack.Stack[deleteStep], nodeId uint, positionInParent uint,
) (encoding.Tuple[K, V], error) {
	var (
		currentNode Node[K, V]
		err         error
	)
	for {
		currentNode, err = t.storage.Get(nodeId)
		if err != nil {
			return encoding.Tuple[K, V]{}, err
		}
		visitedNodes.Push(deleteStep{currentNode.id, positionInParent})
		if currentNode.leaf {
			break
		}
		// currentNode.children is always a and thus non-zero
		positionInParent = uint(len(currentNode.children) - 1)
		nodeId = currentNode.children[positionInParent]
	}

	var value encoding.Tuple[K, V]
	value, currentNode.values = popLast(currentNode.values)
	return value, t.storage.Persist(currentNode)
}

func (t *Tree[K, V]) merge(
	middleValuePosition uint, left, right Node[K, V], parent *Node[K, V], parentIsRoot bool,
) error {
	// middleValuePosition equals position of the left child
	_, parent.children = pop(parent.children, middleValuePosition+1)
	var middleValue encoding.Tuple[K, V]
	middleValue, parent.values = pop(parent.values, middleValuePosition)
	left.values = append(append(left.values, middleValue), right.values...)
	left.children = append(left.children, right.children...)
	if parentIsRoot && len(parent.values) == 0 {
		// parent is root with no stored value, left is the new root
		err := t.storage.Remove(left.id)
		if err != nil {
			return err
		}
		left.id = parent.id
		err = t.updateDepth(t.depth - 1)
		if err != nil {
			return err
		}
	} else {
		err := t.storage.Persist(*parent)
		if err != nil {
			return err
		}
	}
	if err := t.storage.Remove(right.id); err != nil {
		return err
	}
	return t.storage.Persist(left)
}

type deleteStep struct {
	visitedNode, positionInParent uint
}

func (t *Tree[K, V]) balanceTreeAfterDelete(path *stack.Stack[deleteStep]) error {
	if path.Count() <= 1 {
		// we don't need to balance tree with only root node
		return nil
	}
	toCheck := path.Pop()
	node, err := t.storage.Get(toCheck.visitedNode)
	if err != nil {
		return err
	}
	for {
		if uint(len(node.values))+1 >= t.a {
			//node does not need to be fixed, we can stop back tracing
			return nil
		}

		parentStep := path.Pop()
		parent, err := t.storage.Get(parentStep.visitedNode)
		if err != nil {
			return err
		}
		parentIsRoot := path.Empty()
		if toCheck.positionInParent == 0 {
			// there is no sibling on the left, we must choose right sibling
			rightSiblingPosition := toCheck.positionInParent + 1
			sibling, err := t.storage.Get(parent.children[rightSiblingPosition])
			if err != nil {
				return err
			}
			if sibling.values.count() >= t.a {
				// we can borrow value from brother
				var (
					valueFromSibling, valueFromParent encoding.Tuple[K, V]
					childFromSibling                  uint
				)
				valueFromSibling, sibling.values = popFirst(sibling.values)
				// position of middle value is equal to index of left node
				valueFromParent = swap(parent.values, toCheck.positionInParent, valueFromSibling)
				node.values = append(node.values, valueFromParent)
				if !node.leaf {
					childFromSibling, sibling.children = popFirst(sibling.children)
					node.children = append(node.children, childFromSibling)
				}
				if err := persistMultiple(t.storage, sibling, parent, node); err != nil {
					return err
				}
			} else {
				err := t.merge(toCheck.positionInParent, node, sibling, &parent, parentIsRoot)
				if err != nil {
					return err
				}
			}
		} else {
			leftSiblingPosition := toCheck.positionInParent - 1
			sibling, err := t.storage.Get(parent.children[leftSiblingPosition])
			if err != nil {
				return err
			}
			if sibling.values.count() >= t.a {
				// we can borrow value from brother
				var valueFromSibling, valueFromParent encoding.Tuple[K, V]
				valueFromSibling, sibling.values = popLast(sibling.values)
				// position of middle value is equal to index of left node
				valueFromParent = swap(parent.values, leftSiblingPosition, valueFromSibling)
				node.values = prepend(node.values, valueFromParent)
				if !node.leaf {
					var childFromSibling uint
					childFromSibling, sibling.children = popLast(sibling.children)
					node.children = prepend(node.children, childFromSibling)
				}
				if err := persistMultiple(t.storage, sibling, parent, node); err != nil {
					return err
				}
			} else {
				err := t.merge(toCheck.positionInParent-1, sibling, node, &parent, parentIsRoot)
				if err != nil {
					return err
				}
			}
		}
		node = parent
		toCheck = parentStep
		if parentIsRoot {
			// root can only be merged with its children, no additional steps are needed
			return nil
		}
	}
}
