// 账本约束数据结构定义
package ledger

// 区块基础操作
type BlockHandle interface {
	GetProposer() []byte
	GetHeight() int64
	GetBlockid() []byte
	GetConsensusStorage() ([]byte, error)
	GetTimestamp() int64
	SetItem(item string, value interface{}) error
	MakeBlockId() ([]byte, error)
	GetPreHash() []byte
	GetNextHash() []byte
	GetPublicKey() string
	GetSign() []byte
	GetTxIDs() []string
	GetInTrunk() bool
}

type SnapshotReader interface {
	Get(bucket string, key []byte) ([]byte, error)
}

type XReader interface {
	Get(bucket string, key []byte) (*VersionedData, error)
	Select(bucket string, startKey []byte, endKey []byte) (XIterator, error)
	//从区块缓存中读取数据信息
	GetUncommited(bucket string, key []byte) (*VersionedData, error)
}

// XIterator iterates over key/value pairs in key order
type XIterator interface {
	Key() []byte
	Value() *VersionedData
	Next() bool
	Error() error
	Close()
}

type PureData struct {
	Bucket string
	Key    []byte
	Value  []byte
}

func (t *PureData) GetBucket() string {
	if t == nil {
		return ""
	}
	return t.Bucket
}

func (t *PureData) GetKey() []byte {
	if t == nil {
		return nil
	}
	return t.Key
}

func (t *PureData) GetValue() []byte {
	if t == nil {
		return nil
	}
	return t.Value
}

type VersionedData struct {
	PureData  *PureData
	RefTxid   []byte
	RefOffset int32
}

func (t *VersionedData) GetPureData() *PureData {
	if t == nil {
		return nil
	}
	return t.PureData
}

func (t *VersionedData) GetRefTxid() []byte {
	if t == nil {
		return nil
	}
	return t.RefTxid
}

func (t *VersionedData) GetRefOffset() int32 {
	if t == nil {
		return 0
	}
	return t.RefOffset
}
