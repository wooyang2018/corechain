package mock

import (
	"errors"
	"fmt"

	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/ledger"
)

type FakeSandBox struct {
	storage map[string]map[string][]byte
}

func (s *FakeSandBox) Get(bucket string, key []byte) ([]byte, error) {
	if _, ok := s.storage[bucket]; !ok {
		return nil, nil
	}
	return s.storage[bucket][utils.F(key)], nil
}

func (s *FakeSandBox) SetContext(bucket string, key, value []byte) {
	if _, ok := s.storage[bucket]; ok {
		s.storage[bucket][utils.F(key)] = value
		return
	}
	addition := make(map[string][]byte)
	addition[utils.F(key)] = value
	s.storage[bucket] = addition
}

type FReaderItem struct {
	Bucket string
	Key    []byte
	Value  []byte
}

type FakeXMReader map[string]FReaderItem

func NewFakeXMReader() FakeXMReader {
	a := make(map[string]FReaderItem)
	return a
}

func (r FakeXMReader) Get(bucket string, key []byte) (*ledger.VersionedData, error) {
	item, ok := r[string(key)]
	if !ok {
		return nil, nil
	}
	return &ledger.VersionedData{
		PureData: &ledger.PureData{
			Bucket: item.Bucket,
			Key:    item.Key,
			Value:  item.Value,
		},
	}, nil
}

func (r *FakeXMReader) Select(bucket string, startKey []byte, endKey []byte) (ledger.XIterator, error) {
	return nil, nil
}

func (r *FakeXMReader) GetUncommited(bucket string, key []byte) (*ledger.VersionedData, error) {
	return nil, errors.New("not support")
}

type FakeLedger struct {
	ledgerSlice   []*FakeBlock
	ledgerMap     map[string]*FakeBlock
	consensusConf []byte
	sandbox       *FakeSandBox
	fakeReader    FakeXMReader
}

func NewFakeLedger(conf []byte) *FakeLedger {
	a := &FakeSandBox{
		storage: make(map[string]map[string][]byte),
	}
	l := &FakeLedger{
		ledgerSlice:   []*FakeBlock{},
		ledgerMap:     map[string]*FakeBlock{},
		consensusConf: conf,
		sandbox:       a,
	}
	l.fakeReader = NewFakeXMReader()
	for i := 0; i < 3; i++ {
		l.Put(NewFakeBlock(i))
	}
	return l
}

func (l *FakeLedger) VerifyMerkle(ledger.BlockHandle) error {
	return nil
}

func (l *FakeLedger) GetGenesisConsensusConf() []byte {
	return l.consensusConf
}

func (l *FakeLedger) Put(block *FakeBlock) error {
	l.ledgerSlice = append(l.ledgerSlice, block)
	id := fmt.Sprintf("%x", block.Blockid)
	l.ledgerMap[id] = block
	return nil
}

func (l *FakeLedger) QueryBlockHeader(blockId []byte) (ledger.BlockHandle, error) {
	id := fmt.Sprintf("%x", blockId)
	if _, ok := l.ledgerMap[id]; !ok {
		return nil, errors.New("not found")
	}
	return l.ledgerMap[id], nil
}

func (l *FakeLedger) QueryBlockHeaderByHeight(height int64) (ledger.BlockHandle, error) {
	if height < 0 {
		return nil, blockSetItemErr
	}
	if int(height) > len(l.ledgerSlice)-1 {
		return nil, blockSetItemErr
	}
	return l.ledgerSlice[height], nil
}

func (l *FakeLedger) GetConsensusConf() ([]byte, error) {
	return l.consensusConf, nil
}

func (l *FakeLedger) GetTipBlock() ledger.BlockHandle {
	if len(l.ledgerSlice) == 0 {
		return nil
	}
	return l.ledgerSlice[len(l.ledgerSlice)-1]
}

func (l *FakeLedger) QueryTipBlockHeader() ledger.BlockHandle {
	return l.GetTipBlock()
}

func (l *FakeLedger) GetTipXMSnapshotReader() (ledger.SnapshotReader, error) {
	return l.sandbox, nil
}

func (l *FakeLedger) CreateSnapshot(blkId []byte) (ledger.XReader, error) {
	return &l.fakeReader, nil
}

func (l *FakeLedger) SetSnapshot(bucket string, key []byte, value []byte) {
	l.fakeReader[string(key)] = FReaderItem{
		Bucket: bucket,
		Key:    key,
		Value:  value,
	}
}

func (l *FakeLedger) GetTipSnapshot() (ledger.XReader, error) {
	return nil, nil
}

func (l *FakeLedger) SetConsensusStorage(height int, s []byte) {
	if len(l.ledgerSlice)-1 < height {
		return
	}
	l.ledgerSlice[height].ConsensusStorage = s
}
