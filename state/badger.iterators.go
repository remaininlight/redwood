package state

import (
	"github.com/dgraph-io/badger/v3"
)

type dbIterator struct {
	iter           *badger.Iterator
	tx             *badger.Txn
	absRootKeypath Keypath
	scanPrefix     Keypath
	rootItem       *badger.Item
	rootNode       *DBNode
	iterNode       *DBNode
	activeIterator *activeIterator
}

// Ensure that dbIterator implements the Iterator interface
var _ Iterator = (*dbIterator)(nil)

func newIteratorFromBadgerIterator(iter *badger.Iterator, absRootKeypath Keypath, rootNode *DBNode) *dbIterator {
	scanPrefix := rootNode.addKeyPrefix(absRootKeypath)
	if len(scanPrefix) != len(rootNode.keyPrefix) {
		scanPrefix = append(scanPrefix, KeypathSeparator[0])
	}
	return &dbIterator{
		iter:           iter,
		tx:             rootNode.tx,
		absRootKeypath: absRootKeypath,
		scanPrefix:     scanPrefix,
		rootNode:       rootNode,
		iterNode:       &DBNode{tx: rootNode.tx, keyPrefix: rootNode.keyPrefix, activeIterator: rootNode.activeIterator},
		activeIterator: rootNode.activeIterator,
	}
}

func (iter *dbIterator) RootKeypath() Keypath {
	return iter.absRootKeypath
}

func (iter *dbIterator) Rewind() {
	rootItem, err := iter.tx.Get(iter.rootNode.addKeyPrefix(iter.absRootKeypath))
	if err == badger.ErrKeyNotFound || err != nil { // Just being explicit here
		// Ignore the root.  Just start iterating from the first actual iterator keypath
		iter.rootItem = nil
		iter.iter.Seek(iter.scanPrefix)
		if !iter.iter.ValidForPrefix(iter.scanPrefix) {
			return
		}
		iter.setNode(iter.iter.Item())
		return
	}
	iter.rootItem = rootItem
	iter.setNode(rootItem)
}

func (iter *dbIterator) SeekTo(relKeypath Keypath) {
	absKeypath := iter.absRootKeypath.Push(relKeypath)
	iter.seekTo(absKeypath)
}

func (iter *dbIterator) seekTo(absKeypath Keypath) {
	if absKeypath.Equals(iter.absRootKeypath) {
		iter.Rewind()
		return
	}
	iter.rootItem = nil
	iter.iter.Seek(iter.rootNode.addKeyPrefix(absKeypath))
	if !iter.iter.ValidForPrefix(iter.scanPrefix) {
		return
	}
	iter.setNode(iter.iter.Item())
}

func (iter *dbIterator) Next() {
	if iter.rootItem != nil {
		iter.rootItem = nil
		iter.iter.Seek(iter.scanPrefix)
		if !iter.Valid() {
			return
		}
		if len(iter.rootNode.rmKeyPrefix(iter.scanPrefix)) == 0 {
			iter.iter.Next()
		}
	} else {
		iter.iter.Next()
	}
	if !iter.Valid() {
		return
	}
	iter.setNode(iter.iter.Item())
}

func (iter *dbIterator) setNode(item *badger.Item) {
	iter.iterNode.rootKeypath = item.KeyCopy(iter.iterNode.rootKeypath)
	iter.iterNode.rootKeypath = iter.rootNode.rmKeyPrefix(iter.iterNode.rootKeypath)
	if len(iter.iterNode.rootKeypath) == 0 {
		iter.iterNode.rootKeypath = nil
	}
}

func (iter *dbIterator) Node() Node {
	if !iter.Valid() {
		return nil
	}
	return iter.iterNode
}

func (iter *dbIterator) NodeCopy() Node {
	if !iter.Valid() {
		return nil
	}
	cp := *iter.iterNode
	return &cp
}

func (iter *dbIterator) Valid() bool {
	if iter.rootItem != nil {
		return true
	}
	return iter.iter.ValidForPrefix(iter.scanPrefix)
}

func (iter *dbIterator) Close() {
	iter.iter.Close()
	iter.activeIterator.iter = nil
}

type dbChildIterator struct {
	Iterator
	numAbsKeypathParts int
}

func newChildIteratorFromBadgerIterator(iter *badger.Iterator, absRootKeypath Keypath, rootNode *DBNode) *dbChildIterator {
	dbIter := newIteratorFromBadgerIterator(iter, absRootKeypath, rootNode)
	return &dbChildIterator{
		Iterator:           dbIter,
		numAbsKeypathParts: dbIter.absRootKeypath.NumParts(),
	}
}

func (iter *dbChildIterator) Next() {
	if !iter.Iterator.Valid() {
		return
	}
	iter.Iterator.Next()
	for ; iter.Iterator.Valid(); iter.Iterator.Next() {
		node := iter.Iterator.Node()
		numParts := node.Keypath().NumParts()
		if numParts == iter.numAbsKeypathParts+1 {
			return
		}
	}
}

func (iter *dbChildIterator) Rewind() {
	iter.Iterator.Rewind()
	iter.Next()
}

type reusableIterator struct {
	Iterator
	absOriginalKeypath Keypath
	rootNode           *DBNode
	badgerIter         *badger.Iterator
}

func newReusableIterator(originalIterator Iterator, absRootKeypath Keypath, rootNode *DBNode) Iterator {
	var badgerIter *badger.Iterator
	switch oi := originalIterator.(type) {
	case *dbIterator:
		badgerIter = oi.iter
	case *dbChildIterator:
		badgerIter = oi.Iterator.(*dbIterator).iter
	default:
		panic("you can only use a reusableIterator with a dbIterator or a dbChildIterator")
	}
	var absOriginalKeypath Keypath
	if originalIterator.Node() == nil {
		originalIterator.Rewind()
	} else {
		absOriginalKeypath = originalIterator.Node().Keypath()
	}
	return &reusableIterator{
		Iterator:           newIteratorFromBadgerIterator(badgerIter, absRootKeypath, rootNode),
		absOriginalKeypath: absOriginalKeypath,
		rootNode:           rootNode,
		badgerIter:         badgerIter,
	}
}

func (ri *reusableIterator) Close() {
	ri.badgerIter.Seek(ri.rootNode.addKeyPrefix(ri.absOriginalKeypath))
}

type dbDepthFirstIterator struct {
	iter           *badger.Iterator
	absRootKeypath Keypath
	scanPrefix     Keypath
	tx             *badger.Txn
	rootNode       *DBNode
	iterNode       *DBNode
	rootItem       *badger.Item
	done           bool
}

// Ensure that dbDepthFirstIterator implements the Iterator interface
var _ Iterator = (*dbDepthFirstIterator)(nil)

func (iter *dbDepthFirstIterator) RootKeypath() Keypath {
	return iter.absRootKeypath
}

func (iter *dbDepthFirstIterator) Valid() bool {
	if iter.rootItem != nil {
		return true
	} else if iter.done {
		return false
	}
	return iter.iter.ValidForPrefix(iter.scanPrefix)
}

func (iter *dbDepthFirstIterator) setNode(item *badger.Item) {
	iter.iterNode.rootKeypath = item.KeyCopy(iter.iterNode.rootKeypath)
	iter.iterNode.rootKeypath = iter.rootNode.rmKeyPrefix(iter.iterNode.rootKeypath)
	if len(iter.iterNode.rootKeypath) == 0 {
		iter.iterNode.rootKeypath = nil
	}
}

func (iter *dbDepthFirstIterator) Rewind() {
	iter.rootItem = nil
	iter.done = false
	iter.iter.Seek(append(iter.scanPrefix, byte(0xff)))
	iter.syncAfterJump()
}

func (iter *dbDepthFirstIterator) SeekTo(relKeypath Keypath) {
	absKeypath := iter.absRootKeypath.Push(relKeypath)
	iter.seekTo(absKeypath)
}

func (iter *dbDepthFirstIterator) seekTo(absKeypath Keypath) {
	iter.iter.Seek(iter.rootNode.addKeyPrefix(absKeypath))
	iter.syncAfterJump()
}

func (iter *dbDepthFirstIterator) Node() Node {
	if !iter.Valid() {
		return nil
	}
	return iter.iterNode
}

func (iter *dbDepthFirstIterator) NodeCopy() Node {
	if !iter.Valid() {
		return nil
	}
	cp := *iter.iterNode
	return &cp
}

func (iter *dbDepthFirstIterator) Next() {
	if iter.done {
		return
	} else if iter.rootItem != nil {
		iter.rootItem = nil
		iter.done = true
		return
	}
	iter.iter.Next()
	iter.syncAfterJump()
}

func (iter *dbDepthFirstIterator) syncAfterJump() {
	if !iter.iter.Valid() {
		iter.rootItem = nil
		iter.done = true
		return
	}

	item := iter.iter.Item()
	kp := iter.rootNode.rmKeyPrefix(item.Key())
	if kp.Equals(iter.absRootKeypath) {
		item := iter.iter.Item()
		iter.setNode(item)
		iter.rootItem = item
		iter.done = false

	} else if iter.iter.ValidForPrefix(iter.scanPrefix) {
		iter.setNode(item)
		iter.done = false

	} else {
		iter.rootItem = nil
		iter.done = true
	}
}

func (iter *dbDepthFirstIterator) Close() {
	iter.iter.Close()
}
