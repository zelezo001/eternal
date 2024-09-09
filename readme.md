[//]: # (# Eternal)

Package providing implementation of configurable (a,b)-tree

## Installation

```bash 
go get -u github.com/zelezo001/eternal
```

## Usage

Firstly import encoding and eternal package 
```go
import "github.com/zelezo001/eternal/encoding"
import "github.com/zelezo001/eternal"

```
and create serializer for your key and value types. (see encoding package for more nuanced control over keys)
```go
type (
    KeyType uint
    ValueType struct {
		Name string `eternal:"size=10"`
    }
)
keySerializer := encoding.CreateForPrimitive[KeyType]()
valueSerializer, err := encoding.Create[ValueType]()
if err != nil { 
	// handle err
}
```
Then prepare file where data will be stored and create persistent storage
```go
file, err := os.OpenFile("storage.eth", os.O_RDWR|os.O_CREATE, 0644)
if err != nil {
 // handle err
}
const a, b = 2, 3
// set same as disk where file is stored for optimal disk operations
const blockSize int64 = 256 
storage, err := ethernal.NewPersistentStorage[KeyType, ValueType](a, b, blockSize, file, keySerializer, valueSerializer)
if err != nil {
// handle err
}
```
Finally, create tree with prepared storage.
```go
tree := eternal.NewTree[KeyType, ValueType](a,b, storage)

```

Now you store, retrieve or delete values.

```go

value, err := tree.Get(12) // finds value with key 12
if err != nil {
// handle err and check for ErrNotFound error
}

err := tree.Delete(12) // deletes value with key 12 if exists
if err != nil {
// handle err
}

value = ValueType{Name: "John"}
err := tree.Insert(12, value) // insert value ValueType{Name: "John"} with key 12
if err != nil {
// handle err
}

```

Don't forget to close storage when your program ends.
```go
err = storage.Close() 
if err != nil {
// handle err
}
```

### Errors 
Only expected error returned from tree is `ErrNotFound`, other errors mean something went wrong with persistence layer.
Unfortunately package does not (yet) provide way to recover from them.

## Usage pitfalls 
Beware that due to serialization to file and address alignment all values must have fixed size and order. 
Changing order/config of fields in serialized value or using strings/slices over declared length can have undefined behavior.