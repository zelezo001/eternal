Eternal data persistence format
---

This document describes format in which eternal stores information about its (a,b)-tree in persistent storage.

## Encoding

If not mentioned otherwise, all numbers are encoded in big endian

## File composition

Eternal file consists of three parts:

1. Header
2. Tree metadata
3. Nodes

### Header

Main purpose of header is to prevent corruption of data, either when system/version of eternal changes or
when serialization strategy changes.

Header occupies first 98 bytes of file and is consists of

| Range       | 0-6                        | 7-8                       | 9-16                                      | 17-80            | 81                      | 82-89       | 90-97       |
|-------------|----------------------------|---------------------------|-------------------------------------------|------------------|-------------------------|-------------|-------------|
| Description | "eternal" encoded as bytes | version of eternal format | block size provided when file was created | schema signature | system bit size - 32/64 | A parameter | B parameter |

### Tree metadata

Tree metadata consists of two values depth and freeId. They are both of type uint and their size is system dependant.
They are stored immediately after header and on 64-bit system their alignment is:

| Range       | 0-3   | 4-7    |
|-------------|-------|--------|
| Description | depth | freeId |

Depth indicates depth of stored (a,b)-tree.

FreeId points to first allocated but free node id. Zero value means there is no such node present.

### Node data

First byte of every node indicated if node is used in the tree or if it's free to be assigned.
Remaining bytes contain values and ids of child nodes.

#### Unused nodes

In unused node first 8 (or 4 for 32-bit systems) bytes indicated next free id. This forms chain of free ids
which help fill unused blocks in the file. Remaining bytes are not used.

#### Alignment

When stored nodes are padded to match provided block size, either to smallest multiple they can fit to, or to
smallest `blockSize/2^n` they can fit to. Their first byte is then located at `paddedSize * nodeId + nodeDataStart`.
First node is stored immediately after tree metadata.
