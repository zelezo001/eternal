package eternal

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"math"
	"math/bits"
	"os"
	"slices"

	"github.com/zelezo001/eternal/encoding"
)

type (
	identifier = [7]byte
	signature  = [64]byte
	version    = int16

	Header struct {
		Identifier identifier
		Version    version
		BlockSize  int64
		Signature  signature
		System     byte // 64/32
		A, B       uint64
	}

	File interface {
		io.ReadWriteSeeker
		io.Closer
	}
)

const (
	rootId         uint    = 0
	currentVersion version = 1
	noFreeId               = 0
)

var eternalIdentifier = identifier{'e', 't', 'e', 'r', 'n', 'a', 'l'}

func checkHeader(header Header, schemaSignature signature, a, b uint) error {
	if header.Identifier == eternalIdentifier {
		return errors.New("file is not eternal data file")
	}
	if header.Version != currentVersion {
		return fmt.Errorf("data file with version %d is not compatible with current version %d", header.Version,
			currentVersion)
	}
	if schemaSignature != header.Signature {
		return errors.New("signature between current data type and data file differs")
	}
	if header.A != uint64(a) || header.B != uint64(b) {
		return fmt.Errorf("data file was created for (%d,%d)-tree but current tree is (%d,%d)-tree", header.A, header.B,
			a, b)
	}
	if bits.UintSize != uint(header.System) {
		return fmt.Errorf("data file was created with %d bits uint, but current system uses %d bits uint",
			header.System, bits.UintSize)
	}
	return nil
}

var (
	headerSerializer encoding.Serializer[Header]
	boolSerializer   = encoding.CreateForPrimitive[bool]()
	uintSerializer   = encoding.CreateForPrimitive[uint]()
)

var _ NodeStorage[string, any] = &PersistentStorage[string, any]{}

func init() {
	var err error
	headerSerializer, err = encoding.Create[Header]()
	if err != nil {
		panic(fmt.Errorf("could not create Header serializer: %w", err))
	}
}

func (p *PersistentStorage[K, V]) loadMetadata() error {
	_, err := p.file.Seek(int64(headerSerializer.Size()), io.SeekStart)
	if err != nil {
		return err
	}
	var metaBytes = make([]byte, uintSerializer.Size()*2)
	_, err = p.file.Read(metaBytes)
	if err != nil {
		return err
	}
	p.depth = uintSerializer.Deserialize(metaBytes)
	p.freeId = uintSerializer.Deserialize(metaBytes[uintSerializer.Size():])
	return nil
}

// checkFile
// Checks if file is compatible and set its offset after header. If file is empty, checkFile innit it.
func (p *PersistentStorage[K, V]) checkFile(blockSize int64) error {
	_, err := p.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	schemaSignature := p.valuesSerializer.Signature()
	headerBytes := make([]byte, headerSerializer.Size())
	readHeaderBytes, err := p.file.Read(headerBytes)
	if err == nil {
		header := headerSerializer.Deserialize(headerBytes)
		if err := checkHeader(header, schemaSignature, p.a, p.b); err != nil {
			return fmt.Errorf("header in provided file is not valid: %w", err)
		}
		return p.loadMetadata()
	} else if readHeaderBytes != 0 || !errors.Is(err, io.EOF) {
		return fmt.Errorf("could not read header from data file: %w", err)
	}
	// file is empty, we must set default values
	header := Header{
		Identifier: eternalIdentifier,
		Version:    currentVersion,
		BlockSize:  blockSize,
		Signature:  schemaSignature,
		A:          uint64(p.a),
		B:          uint64(p.b),
		System:     bits.UintSize,
	}
	_, err = p.file.Write(headerSerializer.Serialize(header))
	if err != nil {
		return err
	}
	err = p.SetDepth(1)
	if err != nil {
		return err
	}
	err = p.updateFreeId(noFreeId)
	if err != nil {
		return err
	}
	return p.Persist(Node[K, V]{
		id:       rootId,
		values:   make(values[K, V], 0),
		children: make([]uint, 0),
		leaf:     true,
	})
}

func NewPersistentStorage[K cmp.Ordered, V any](
	a, b uint, blockSize int64, file *os.File, keySerializer encoding.Serializer[K],
	valueSerializer encoding.Serializer[V],
) (
	*PersistentStorage[K, V], error,
) {
	if b >= math.MaxUint32 {
		return nil, errors.New("b parameter must be less than max uint32")
	}
	tupleSerializer := encoding.CreateForTuple(keySerializer, valueSerializer)
	valuesEncoder, err := encoding.CreateSliceForSerializer(tupleSerializer, uint32(b-1))
	if err != nil {
		return nil, fmt.Errorf("could not create serializer for encoding of values: %w", err)
	}
	childrenEncoder, err := encoding.CreateForSlice[[]uint, uint](uint32(b))
	if err != nil {
		return nil, fmt.Errorf("could not create serializer for encoding of child ids: %w", err)
	}

	nodeSize := valuesEncoder.Size() + childrenEncoder.Size()
	nodeSize = max(nodeSize, uintSerializer.Size()) + boolSerializer.Size()

	// we want nodes to be aligned with paddedNodeSize, so we can easily translate between address and id
	metadataSize := uintSerializer.Size() * 2
	paddedNodeSize := calculatePaddedNodeSize(int64(nodeSize), blockSize)

	depthAddress := int64(headerSerializer.Size())
	freeIdAddress := depthAddress + int64(uintSerializer.Size())
	// we are loading tree which is already initialized, we have to load tree metadata
	metaBytes := make([]byte, uintSerializer.Size()*2)
	_, err = file.Read(metaBytes)
	if err != nil {
		return nil, err
	}
	depth := uintSerializer.Deserialize(metaBytes)
	freeId := uintSerializer.Deserialize(metaBytes[uintSerializer.Size():])

	return &PersistentStorage[K, V]{
		nodeSize:           int64(nodeSize),
		file:               file,
		depth:              depth,
		freeId:             freeId,
		depthAddress:       depthAddress,
		freeIdAddress:      freeIdAddress,
		baseNodeAddress:    int64(metadataSize + headerSerializer.Size()),
		valuesSerializer:   valuesEncoder,
		childrenSerializer: childrenEncoder,
		paddedNodeSize:     paddedNodeSize,
		a:                  a,
		b:                  b,
	}, nil
}

type PersistentStorage[K cmp.Ordered, V any] struct {
	nodeSize                    int64
	paddedNodeSize              int64
	a, b                        uint
	file                        *os.File
	depth, freeId               uint  // id which is not occupied in file but is allocated
	depthAddress, freeIdAddress int64 // addresses for tree metadata
	baseNodeAddress             int64 // part of file where nodes are stored
	valuesSerializer            encoding.Serializer[[]encoding.Tuple[K, V]]
	childrenSerializer          encoding.Serializer[[]uint]
}

func (p *PersistentStorage[K, V]) Close() error {
	return p.file.Close()
}

func (p *PersistentStorage[K, V]) GetRoot() (Node[K, V], error) {
	return p.Get(rootId)
}

func (p *PersistentStorage[K, V]) GetDepth() uint {
	return p.depth
}

func (p *PersistentStorage[K, V]) SetDepth(depth uint) error {
	p.depth = depth
	_, err := p.file.WriteAt(uintSerializer.Serialize(p.depth), p.depthAddress)
	return err
}

var ErrMissingNode = errors.New("node not found")

func (p *PersistentStorage[K, V]) Get(id uint) (Node[K, V], error) {
	offset := p.idToOffset(id)
	_, err := p.file.Seek(offset, io.SeekStart)
	if err != nil {
		return Node[K, V]{}, err
	}
	var nodeData = make([]byte, p.nodeSize)
	_, err = p.file.Read(nodeData)
	if err != nil {
		return Node[K, V]{}, err
	}
	empty := boolSerializer.Deserialize(nodeData)
	if empty {
		return Node[K, V]{}, ErrMissingNode
	}
	nodeData = nodeData[boolSerializer.Size():]
	values := p.valuesSerializer.Deserialize(nodeData)
	children := p.childrenSerializer.Deserialize(nodeData[p.valuesSerializer.Size():])
	return Node[K, V]{
		id:       id,
		values:   slices.Grow(values, int(p.b)),
		children: slices.Grow(children, int(p.b+1)),
		leaf:     len(children) == 0,
	}, nil
}

func (p *PersistentStorage[K, V]) Persist(node Node[K, V]) error {
	offset := p.idToOffset(node.id)
	_, err := p.file.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	_, err = p.file.Write(boolSerializer.Serialize(true))
	if err != nil {
		return err
	}
	_, err = p.file.Write(p.valuesSerializer.Serialize(node.values))
	if err != nil {
		return err
	}
	_, err = p.file.Write(p.childrenSerializer.Serialize(node.children))
	return err
}

func (p *PersistentStorage[K, V]) Remove(id uint) error {
	if id == rootId {
		return errors.New("cannot remove root")
	}
	offset := p.idToOffset(id)
	_, err := p.file.Seek(offset, io.SeekStart)
	if err != nil {
		return err
	}
	// lazy delete, proper cleanup will be done during defragmentation or when id is claimed by a new node
	_, err = p.file.Write(boolSerializer.Serialize(false))
	if err != nil {
		return err
	}

	return err
}

func (p *PersistentStorage[K, V]) NewId() (uint, error) {
	if p.freeId == noFreeId {
		// no free space is present in file, we must enlarge file
		address, err := p.file.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, err
		}
		newId := uint((address - p.baseNodeAddress) / p.paddedNodeSize)
		_, err = p.file.Write(boolSerializer.Serialize(false))
		if err != nil {
			return 0, err
		}
		_, err = p.file.Write(make([]byte, p.paddedNodeSize-int64(boolSerializer.Size())))
		if err != nil {
			return 0, err
		}
		return newId, nil
	}
	offset := p.idToOffset(p.freeId)
	_, err := p.file.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	var freeNodeData = make([]byte, boolSerializer.Size()+uintSerializer.Size())
	_, err = p.file.Read(freeNodeData)
	if err != nil {
		return 0, err
	}
	if boolSerializer.Deserialize(freeNodeData) {
		return 0, fmt.Errorf("node with id %d should be free but isn't", p.freeId)
	}
	freeId := p.freeId
	nextFreeId := uintSerializer.Deserialize(freeNodeData[boolSerializer.Size():])
	return freeId, p.updateFreeId(nextFreeId)
}

func (p *PersistentStorage[K, V]) updateFreeId(id uint) error {
	p.freeId = id
	_, err := p.file.WriteAt(uintSerializer.Serialize(id), p.freeIdAddress)
	return err
}

func (p *PersistentStorage[K, V]) idToOffset(id uint) int64 {
	return p.baseNodeAddress + int64(id)*p.paddedNodeSize
}

func calculatePaddedNodeSize(nodeSize, blockSize int64) int64 {
	switch {
	case blockSize == 1:
		return nodeSize
	case nodeSize >= blockSize:
		blocksPerNode := nodeSize / blockSize
		// check if we need one extra block for padding
		if remaining := nodeSize % blockSize; remaining > 0 {
			blocksPerNode++
		}
		nodeSize := blocksPerNode * blockSize
		return nodeSize
	default: // nodeSize < blockSize
		paddedNodeSize := blockSize
		for paddedNodeSize/nodeSize > 1 {
			paddedNodeSize /= 2
		}
		return paddedNodeSize
	}
}
