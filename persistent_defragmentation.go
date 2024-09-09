package eternal

import (
	"errors"
	"io"

	"github.com/zelezo001/eternal/encoding"
)

// Defragment
// removes fragmentation in file by rearranging nodes.
// Defragmentation can lead to change in node IDs, so it shouldn't be called in parallel with tree operations
func (p *PersistentStorage[K, V]) Defragment() error {
	if p.freeId == 0 {
		// no free id is present, that means no inner address/id is unoccupied and file is not fragmented
		return nil
	}
	lastAddress, err := p.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	// lastAddress is at the end of the last node
	lastId := uint((lastAddress-p.baseNodeAddress)/p.paddedNodeSize) - 1
	if lastId == 0 {
		// file contain only root defragmentation is not needed
		return nil
	}
	var (
		searchForFreeBlockFrom uint = 1 // block with id = 0 cannot be empty
	)
	freeBlock, err := p.findEmptyBlock(searchForFreeBlockFrom)
	if err != nil {
		return err
	}
	if freeBlock.Second == 0 {
		return errors.New("there should be at least one free block")
	}
	var firstEmptyNodeId uint
	for reorderedNodeId := rootId; reorderedNodeId <= lastId; reorderedNodeId++ {
		reorderedNode, err := p.loadWithoutValues(reorderedNodeId)
		if err != nil {
			if errors.Is(err, ErrMissingNode) {
				// because we reorder nodes from root, ErrMissingNode means no node with id reorderedNodeId or greater
				// is child of node with id < reorderedNodeId and thus is not in the tree
				firstEmptyNodeId = reorderedNodeId
				break
			}
			return err
		}
		for i := 0; i < len(reorderedNode.children); i++ {
			// we only want to move nodes which are after freeBlock, otherwise we would create empty blocks
			// in the already defragmented part
			if freeBlock.First < reorderedNode.children[i] {
				err := p.moveNode(reorderedNode.children[i], freeBlock.First)
				if err != nil {
					if persistErr := p.Persist(reorderedNode); persistErr != nil {
						err = errors.Join(persistErr, err)
					}
					return err
				}
				reorderedNode.children[i] = freeBlock.First
				freeBlock.First++
				if freeBlock.First > freeBlock.Second {
					freeBlock, err = p.findEmptyBlock(freeBlock.Second + 1)
					if err != nil {
						if persistErr := p.Persist(reorderedNode); persistErr != nil {
							err = errors.Join(persistErr, err)
						}
						return err
					}
					if freeBlock.Second == 0 {
						return errors.New("there should be at least one free block")
					}
				}
			}
		}
	}
	// there is no free space in file, we must set free id to noFreeId
	if err := p.updateFreeId(noFreeId); err != nil {
		return err
	}
	// offset of firstEmptyNodeId is equal to final size of defragmented file
	return p.file.Truncate(p.idToOffset(firstEmptyNodeId))
}

func (p *PersistentStorage[K, V]) moveNode(oldId, newId uint) error {
	var node = make([]byte, p.nodeSize)
	_, err := p.file.ReadAt(node, p.idToOffset(oldId))
	if err != nil {
		return err
	}
	_, err = p.file.WriteAt(node, p.idToOffset(newId))
	return nil
}

func (p *PersistentStorage[K, V]) persistWithoutValues(node Node[K, V]) error {
	offset := p.idToOffset(node.id)
	_, err := p.file.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}

	_, err = p.file.Write(boolSerializer.Serialize(true))
	if err != nil {
		return err
	}
	// skip memory where values are stored
	_, err = p.file.Seek(int64(p.valuesSerializer.Size()), io.SeekCurrent)
	if err != nil {
		return err
	}
	_, err = p.file.Write(p.childrenSerializer.Serialize(node.children))
	return nil
}

func (p *PersistentStorage[K, V]) loadWithoutValues(id uint) (Node[K, V], error) {
	offset := p.idToOffset(id)
	_, err := p.file.Seek(offset, io.SeekStart)
	if err != nil {
		return Node[K, V]{}, err
	}
	var emptyData = make([]byte, boolSerializer.Size())
	_, err = p.file.Read(emptyData)
	if err != nil {
		return Node[K, V]{}, err
	}
	empty := boolSerializer.Deserialize(emptyData)
	if empty {
		return Node[K, V]{}, ErrMissingNode
	}
	// skip memory where values are stored
	_, err = p.file.Seek(int64(p.valuesSerializer.Size()), io.SeekCurrent)
	if err != nil {
		return Node[K, V]{}, err
	}
	var childrenData = make([]byte, p.childrenSerializer.Size())
	children := p.childrenSerializer.Deserialize(childrenData)
	return Node[K, V]{
		id:       id,
		values:   nil,
		children: children,
		leaf:     len(children) == 0,
	}, nil
}

func (p *PersistentStorage[K, V]) findEmptyBlock(startAt uint) (encoding.Tuple[uint, uint], error) {
	var freeBlockBeginning uint
	for {
		inUse, err := p.checkIfInUse(startAt)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return encoding.Tuple[uint, uint]{}, nil
			}
			return encoding.Tuple[uint, uint]{}, err
		}
		if !inUse {
			freeBlockBeginning = startAt
			break
		}
		startAt++
	}

	freeBlockEnd := freeBlockBeginning
	for {
		startAt++
		inUse, err := p.checkIfInUse(startAt)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return encoding.Tuple[uint, uint]{}, err
		}
		if inUse {
			break
		}
		freeBlockEnd = startAt
	}

	return encoding.Tuple[uint, uint]{First: freeBlockBeginning, Second: freeBlockEnd}, nil
}

func (p *PersistentStorage[K, V]) checkIfInUse(id uint) (bool, error) {
	var bytes = make([]byte, boolSerializer.Size())
	_, err := p.file.ReadAt(bytes, p.idToOffset(id))
	if err != nil {
		return false, err
	}

	return boolSerializer.Deserialize(bytes), nil
}
