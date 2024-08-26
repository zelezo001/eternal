package eternal

import "eternal/internal/stack"

func (t *tree[K, V]) Delete(key K) error {
	path := stack.NewStack[deleteStep](t.depth)
	root, err := t.storage.GetRoot()
	if err != nil {
		return err
	}
	var (
		currentNode           = root
		positionInParent uint = 0
	)
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
				valueToReplace, err := t.popLargest(path, leftChildId, uint(position))
				if err != nil {
					return err
				}
				currentNode.values[position] = valueToReplace
				break
			}
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

func (t *tree[K, V]) popLargest(
	visitedNodes *stack.Stack[deleteStep], nodeId uint, positionInParent uint,
) (tuple[K, V], error) {
	var (
		currentNode Node[K, V]
		err         error
	)
	for {
		currentNode, err = t.storage.Get(nodeId)
		if err != nil {
			return tuple[K, V]{}, err
		}
		visitedNodes.Push(deleteStep{currentNode.id, positionInParent})
		if currentNode.leaf {
			break
		}
		// currentNode.children is always a and thus non-zero
		positionInParent = uint(len(currentNode.children) - 1)
		nodeId = currentNode.children[positionInParent]
	}
	value := currentNode.values[len(currentNode.values)-1]
	currentNode.values = currentNode.values[:len(currentNode.values)-1]
	return value, nil
}

func (t *tree[K, V]) merge(middleValuePosition uint, left, right Node[K, V], parent *Node[K, V]) error {
	// middleValuePosition equals position of the left child
	_, parent.children = pop(parent.children, middleValuePosition+1)
	var middleValue tuple[K, V]
	middleValue, parent.values = pop(parent.values, middleValuePosition)
	left.values = append(append(left.values, middleValue), right.values...)
	left.children = append(left.children, right.children...)
	if len(parent.values) > 0 {
		err := t.storage.Persist(*parent)
		if err != nil {
			return err
		}
	} else {
		// parent is root with no stored value, left is the new root
		err := t.storage.Remove(left.id)
		if err != nil {
			return err
		}
		left.id = parent.id
		t.depth--
	}
	if err := t.storage.Remove(right.id); err != nil {
		return err
	}
	return t.storage.Persist(left)
}

type deleteStep struct {
	visitedNode, positionInParent uint
}

func (t *tree[K, V]) balanceTreeAfterDelete(path *stack.Stack[deleteStep]) error {
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
			// we fixed tree up to this node
			return nil
		}

		parentStep := path.Pop()
		parent, err := t.storage.Get(parentStep.visitedNode)
		if err != nil {
			return err
		}

		if toCheck.positionInParent == 0 {
			rightSiblingPosition := toCheck.positionInParent + 1
			sibling, err := t.storage.Get(parent.children[rightSiblingPosition])
			if err != nil {
				return err
			}
			if sibling.values.count() >= t.a {
				// we can borrow value from brother
				var (
					valueFromSibling, valueFromParent tuple[K, V]
					childFromSibling                  uint
				)
				valueFromSibling, sibling.values = popFirst(sibling.values)
				childFromSibling, sibling.children = popFirst(sibling.children)
				valueFromParent = swap(parent.values, toCheck.positionInParent, valueFromSibling)
				node.values = append(node.values, valueFromParent)
				node.children = append(node.children, childFromSibling)
				if err := persistMultiple(t.storage, sibling, parent, node); err != nil {
					return err
				}
			} else {
				err := t.merge(toCheck.positionInParent, node, sibling, &parent)
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
				var (
					valueFromSibling, valueFromParent tuple[K, V]
					childFromSibling                  uint
				)
				valueFromSibling, sibling.values = popLast(sibling.values)
				childFromSibling, sibling.children = popLast(sibling.children)
				valueFromParent = swap(parent.values, toCheck.positionInParent, valueFromSibling)
				node.values = prepend(node.values, valueFromParent)
				node.children = prepend(node.children, childFromSibling)
				if err := persistMultiple(t.storage, sibling, parent, node); err != nil {
					return err
				}
			} else {
				err := t.merge(toCheck.positionInParent, sibling, node, &parent)
				if err != nil {
					return err
				}
			}
		}
		node = parent
		if path.Count() <= 1 {
			// root can only be merged with its children, no additional steps are need
		}
	}
}
