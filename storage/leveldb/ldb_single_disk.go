package leveldb

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Open opens an instance of LDB with parameters (ldb path and other options)
func (ldb *LDBDatabase) OpenSingle(path string, options map[string]interface{}) error {
	setDefaultOptions(options)
	cache := options["cache"].(int)
	fds := options["fds"].(int)
	dataPaths := options["dataPaths"].([]string)

	// Open the db and recover any potential corruptions
	// 如果没有配置多盘则按照Single方式打开数据库
	if dataPaths == nil || len(dataPaths) == 0 {
		db, err := leveldb.OpenFile(path, &opt.Options{
			OpenFilesCacheCapacity: fds,
			BlockCacheCapacity:     cache / 2 * opt.MiB,
			WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
			Filter:                 filter.NewBloomFilter(10),
		})
		if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
			return err
		}
		// (Re)check for errors and abort if opening of the db failed
		if err != nil {
			return err
		}
		ldb.fn = path
		ldb.db = db
		return nil
	}

	return fmt.Errorf("multi disk only supported in commercial version")
}
