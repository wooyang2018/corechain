package event

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

// Topic is the factory of event Iterator
type Topic interface {
	// ParseFilter 从指定的bytes buffer反序列化topic过滤器
	// 返回的参数会作为入参传递给NewIterator的filter参数
	ParseFilter(buf []byte) (interface{}, error)

	// MarshalEvent encode event payload returns from Iterator.Data()
	MarshalEvent(x interface{}) ([]byte, error)

	// NewIterator make a new Iterator base on filter
	NewIterator(filter interface{}) (Iterator, error)
}

// blockTopic handles block events
type blockTopic struct {
	chainmg ChainManager
}

var _ Topic = (*blockTopic)(nil)

// NewBlockTopic instances blockTopic from ChainManager
func NewBlockTopic(chainmg ChainManager) *blockTopic {
	return &blockTopic{
		chainmg: chainmg,
	}
}

// NewFilterIterator make a new Iterator base on filter
func (b *blockTopic) NewFilterIterator(pbfilter *protos.BlockFilter) (Iterator, error) {
	filter, err := newBlockFilter(pbfilter)
	if err != nil {
		return nil, err
	}
	return b.newIterator(filter)
}

// ParseFilter 从指定的bytes buffer反序列化topic过滤器
// 返回的参数会作为入参传递给NewIterator的filter参数
func (b *blockTopic) ParseFilter(buf []byte) (interface{}, error) {
	pbfilter := new(protos.BlockFilter)
	err := proto.Unmarshal(buf, pbfilter)
	if err != nil {
		return nil, err
	}

	return pbfilter, nil
}

// MarshalEvent encode event payload returns from Iterator.Data()
func (b *blockTopic) MarshalEvent(x interface{}) ([]byte, error) {
	msg := x.(proto.Message)
	return proto.Marshal(msg)
}

// NewIterator make a new Iterator base on filter
func (b *blockTopic) NewIterator(ifilter interface{}) (Iterator, error) {
	pbfilter, ok := ifilter.(*protos.BlockFilter)
	if !ok {
		return nil, errors.New("bad filter type for block event")
	}
	filter, err := newBlockFilter(pbfilter)
	if err != nil {
		return nil, err
	}
	return b.newIterator(filter)
}

func (b *blockTopic) newIterator(filter *blockFilter) (Iterator, error) {
	blockStore, err := b.chainmg.GetBlockStore(filter.GetBcName())
	if err != nil {
		return nil, err
	}

	var startBlockNum, endBlockNum int64
	if filter.GetRange().GetStart() == "" {
		n, err := blockStore.TipBlockHeight()
		if err != nil {
			return nil, err
		}
		startBlockNum = n
	} else {
		n, err := strconv.ParseInt(filter.GetRange().GetStart(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error %s when parse start block number", err)
		}
		startBlockNum = n
	}

	if filter.GetRange().GetEnd() == "" {
		endBlockNum = -1
	} else {
		n, err := strconv.ParseInt(filter.GetRange().GetEnd(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error %s when parse end block number", err)
		}
		endBlockNum = n
	}

	biter := NewBlockIterator(blockStore, startBlockNum, endBlockNum)
	return &filteredBlockIterator{
		biter:  biter,
		filter: filter,
	}, nil
}
