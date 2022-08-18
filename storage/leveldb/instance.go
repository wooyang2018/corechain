package leveldb

import (
	"errors"
	"fmt"
	"sync"

	"github.com/wooyang2018/corechain/storage"
)

// KVParameter structure for kv instance parameters
type KVParameter struct {
	DBPath                string
	KVEngineType          string
	StorageType           string
	MemCacheSize          int
	FileHandlersCacheSize int
	OtherPaths            []string
	Options               map[string]interface{}
}

const (
	KVEngineType      = "leveldb"
	StorageTypeSingle = "single"
)

var (
	servsMu  sync.RWMutex
	services = make(map[string]NewStorageFunc)
)

type NewStorageFunc func(*KVParameter) (storage.Database, error)

func Register(name string, f NewStorageFunc) {
	servsMu.Lock()
	defer servsMu.Unlock()

	if f == nil {
		panic("storage: Register new func is nil")
	}
	if _, dup := services[name]; dup {
		panic("storage: Register called twice for func " + name)
	}
	services[name] = f
}

func CreateKVInstance(kvParam *KVParameter) (storage.Database, error) {
	servsMu.RLock()
	defer servsMu.RUnlock()

	if f, ok := services[kvParam.KVEngineType]; ok {
		instance, err := f(kvParam)
		if err != nil {
			return nil, fmt.Errorf("get kvInstance fail:%w", err)
		}
		return instance, nil
	}

	return nil, errors.New("get kvInstance fail")
}

// GetDBPath return the value of DBPath
func (param *KVParameter) GetDBPath() string {
	return param.DBPath
}

// GetKVEngineType return the value of KVEngineType
func (param *KVParameter) GetKVEngineType() string {
	return param.KVEngineType
}

// GetStorageType return the value of GetStorageType
func (param *KVParameter) GetStorageType() string {
	return param.StorageType
}

// GetMemCacheSize return the value of MemCacheSize
func (param *KVParameter) GetMemCacheSize() int {
	return param.MemCacheSize
}

// GetFileHandlersCacheSize return the value of FileHandlersCacheSize
func (param *KVParameter) GetFileHandlersCacheSize() int {
	return param.FileHandlersCacheSize
}

// GetOtherPaths return the value of OtherPaths
func (param *KVParameter) GetOtherPaths() []string {
	return param.OtherPaths
}
