package model

import (
	"bytes"
	"sort"

	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/ledger/def"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/storage"
	"google.golang.org/protobuf/proto"
)

// KVEngineType KV storage type
const KVEngineType = "default"

// BucketSeperator separator between bucket and raw key
const BucketSeperator = "/"

// DelFlag delete flag
const DelFlag = "\x00"

func isDelFlag(value []byte) bool {
	return bytes.Equal([]byte(DelFlag), value)
}

// MakeRawKey make key with bucket and raw key
func MakeRawKey(bucket string, key []byte) []byte {
	return makeRawKey(bucket, key)
}

func makeRawKey(bucket string, key []byte) []byte {
	k := append([]byte(bucket), []byte(BucketSeperator)...)
	return append(k, key...)
}

func queryUnconfirmTx(txid []byte, table storage.Database) (*protos.Transaction, error) {
	pbBuf, findErr := table.Get(txid)
	if findErr != nil {
		return nil, findErr
	}
	tx := &protos.Transaction{}
	pbErr := proto.Unmarshal(pbBuf, tx)
	if pbErr != nil {
		return nil, pbErr
	}
	return tx, nil
}

func saveUnconfirmTx(tx *protos.Transaction, batch storage.Batch) error {
	buf, err := proto.Marshal(tx)
	if err != nil {
		return err
	}
	rawKey := append([]byte(def.UnconfirmedTablePrefix), []byte(tx.Txid)...)
	batch.Put(rawKey, buf)
	return nil
}

// 快速对写集合排序
type pdSlice []*ledger.PureData

// newPdSlice new a slice instance for PureData
func newPdSlice(vpd []*ledger.PureData) pdSlice {
	s := make([]*ledger.PureData, len(vpd))
	copy(s, vpd)
	return s
}

// Len length of slice of PureData
func (pds pdSlice) Len() int {
	return len(pds)
}

// Swap swap two pureData elements in a slice
func (pds pdSlice) Swap(i, j int) {
	pds[i], pds[j] = pds[j], pds[i]
}

// Less compare two pureData elements with pureData's key in a slice
func (pds pdSlice) Less(i, j int) bool {
	rawKeyI := makeRawKey(pds[i].GetBucket(), pds[i].GetKey())
	rawKeyJ := makeRawKey(pds[j].GetBucket(), pds[j].GetKey())
	ret := bytes.Compare(rawKeyI, rawKeyJ)
	if ret == 0 {
		// 注: 正常应该无法走到这个逻辑，因为写集合中的key一定是唯一的
		return bytes.Compare(pds[i].GetValue(), pds[j].GetValue()) < 0
	}
	return ret < 0
}

func equal(pd, vpd *ledger.PureData) bool {
	rawKeyI := makeRawKey(pd.GetBucket(), pd.GetKey())
	rawKeyJ := makeRawKey(vpd.GetBucket(), vpd.GetKey())
	ret := bytes.Compare(rawKeyI, rawKeyJ)
	if ret != 0 {
		return false
	}
	return bytes.Equal(pd.GetValue(), vpd.GetValue())
}

// Equal check if two PureData object equal
func Equal(pd, vpd []*ledger.PureData) bool {
	if len(pd) != len(vpd) {
		return false
	}
	pds := newPdSlice(pd)
	vpds := newPdSlice(vpd)
	sort.Sort(pds)
	sort.Sort(vpds)
	for i, v := range pds {
		if equal(v, vpds[i]) {
			continue
		}
		return false
	}
	return true
}
