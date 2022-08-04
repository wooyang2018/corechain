package model

import (
	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/storage"
)

// XIterator data structure for XModel Iterator
type XIterator struct {
	bucket string
	iter   storage.Iterator
	model  *XModel
	value  *ledger.VersionedData
	err    error
}

// Data get data pointer to VersionedData for XIterator
func (di *XIterator) Value() *ledger.VersionedData {
	return di.value
}

// Next check if next element exist
func (di *XIterator) Next() bool {
	ok := di.iter.Next()
	if !ok {
		return false
	}
	version := di.iter.Value()
	verData, err := di.model.fetchVersionedData(di.bucket, string(version))
	if err != nil {
		di.err = err
		return false
	}
	di.value = verData
	return true
}

// Key get key for XIterator
func (di *XIterator) Key() []byte {
	v := di.Value()
	if v == nil {
		return nil
	}
	return v.GetPureData().GetKey()
}

// Error return error info for XIterator
func (di *XIterator) Error() error {
	kverr := di.iter.Error()
	if kverr != nil {
		return kverr
	}
	return di.err
}

// Release release XIterator
func (di *XIterator) Close() {
	di.iter.Release()
	di.value = nil
}
