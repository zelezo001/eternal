# Eternal

## Introduction

Goal of this library is to provide simple persistent key-value storage
written in Go.

## Implementation

Library is build around two main types `eternal.Tree` struct and `eternal.NodeStorage` interface.
`eternal.Tree` contains logic connected to (a,b)-tree and `eternal.NodeStorage` provides way do decouple actual storage
of data logic of `eternal.Tree`

Currently, two implementations of `eternal.NodeStorage` are provided.
`eternal.InMemoryStorage` is simple implementation indented mainly for testing purposes.
`eternal.PersistentStorage` is full-fledged implementation used for longtime usage.

### Tree

(a,b)-tree is tree structure with following rules:

1. root has between 2 and b children
2. inner nodes (nodes which are not root but also not leaves) has between a and b children
3. for every leaf, path to root has same length
4. b is at least a*2-1

In this implementation leaf nodes are not NILs but nodes with zero children.

#### Find

Find operation is implemented by method `eternal.Tree.Get(key)` and works as standard BST search with only difference
being that multiple keys can be stored in one node. In a way keys stored in one node represent inner BST whose
leaves are children of given node.

#### Insert

Insert operation implemented by method `eternal.Tree.Insert(key, value)` has two stages.

Firstly key is inserted into tree. If record with same key is present then value is replaced and insert stops. If key
is not yet present then key-value pair is inserted into corresponding node, which is always leaf, and second stage of
insert
is executed.

In the second stage, we walk tree back to the root from the leaf in which we inserted key-value pair in previous stage.
For every node on the path we check if rule 1. or 2. is not violated. If no rule is violated, then we can stop as
current
node is fine and as we didn't change nodes before us.

Violation of rules 1 or 2 means that current node has b inner values (underflow cannot happen as we
don't delete).
We will fix this by taking middle value from node and splitting current node into two (and also removing it from its
parent). Because split node had exactly b
inner values which is at least 2*a-1 values and because we took middle value out, we now have two nodes with a-1 values.

In the next step we insert middle value we took earlier into parent node together with two new nodes that we created by
spitting current node.

This will fail if we try splitting root as root has no parent, in this scenario we will simply create new root
which will act as a parent of the old root.

In case we split the root of the tree we will also increase the depth of the tree,
otherwise as we added value to parent we potentially again violated one of the rules, we must continue in our path to
the root.

```
Example of splitting node of (2,3)-tree

    (0)             (0,2)
   /   \           /  |  \
  c  (1,2,3) ->   c  (1) (3)
```

#### Delete

Delete operation implemented by method `eternal.Tree.Delete(key)` and also has two stages.

Firstly we try to find value with given key. If no such pair exists, we will simply end as we have nothing to delete.
If we found pair we are looking for in leaf node we will delete it from the node and continue to stage 2.
We cannot remove inner node as it acts as divider between two children, so we will firstly replace it with its greatest
predecessor and then process as we would if we deleted the predecessor.
Greatest predecessor is always in one of leaf nodes, so we can continue to stage 2.

In the stage 2 we will walk back to the root and fix any violations of rules 1 or 2.
If the current node has at least a-1 inner values,we can stop as current node is fine and as we didn't change nodes
before us.

Only possible violation is that the current node as less than a-1 inner values.
There are two possible ways we can fix this.
If the node has left (or right for leftmost node) sibling with more than a-1 inner values, then we can borrow one value
from it.

```
Example of borrowing from right sibling in (2,3)-tree

    (0)             (1)
   /   \            / \
  ()  (1,2) ->   (0)  (2)
```

This will restore all the rules and as we didn't alter number of values in the parent node we can also stop.
If sibling has exactly a-1 inner values, we can merge it with the current node and value between them. As b is at least
a*2 - 1, the new node will have at most b inner values.

```
Example of merging in (2,3)-tree

    (0)            ()
   /   \            |
  ()  (1) ->      (0,1)
```
This can lead to violation of rules 1 or 2 in the parent of current node, so we must continue with checking.
If hit the root and it has zero inner values, we will simply remove it and mark its only child as the new root.  

### PersistentStorage

`eternal.PeristentStorage` implement `eternal.NodeStorage` which stores tree nodes in file using eternal
file format described in `file_format.md`

It also implements `Defragment` function which walks whole file, rearranges nodes and trims file in way that file does
not
contain blocks of unused space.

#### Encoding

Encoding and decoding between bytes form and go values is handled `eternal/encoding` package.

Package is exposing `encoding.Serializer` and for it handful of factory functions intended for various types.

Only structs, scalars, strings, arrays and slices are supported. For variable sized types like slices and strings,
maximal size must be provided.

Internally, `eternal/encoding` works with so-called blueprints, which describe given type.
Creation of blueprint from type is handled by function `handleType`, which maps type with help of `reflection` package.

For struct types tags are used to define variable sized properties.

All int values are encoded in big endian format.

------

##### Sources

https://pruvodce.ucw.cz/