package event

import (
	"encoding/hex"
	"testing"

	"github.com/wooyang2018/corechain/engine/mock"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

func TestRouteBlockTopic(t *testing.T) {
	ledger := mock.NewMockBlockStore()
	block := mock.NewBlockBuilder().Block()
	ledger.AppendBlock(block)

	router := NewRounterFromChainMG(ledger)

	filter := &protos.BlockFilter{
		Range: &protos.BlockRange{
			Start: "0",
		},
	}
	buf, err := proto.Marshal(filter)
	if err != nil {
		t.Fatal(err)
	}
	encfunc, iter, err := router.Subscribe(protos.SubscribeType_BLOCK, buf)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	iter.Next()
	fblock := iter.Data().(*protos.FilteredBlock)

	_, err = encfunc(fblock)
	if err != nil {
		t.Fatal(err)
	}

	if fblock.GetBlockid() != hex.EncodeToString(block.GetBlockid()) {
		t.Fatalf("block not equal, expect %x got %s", block.GetBlockid(), fblock.GetBlockid())
	}
}

func TestRouteBlockTopicRaw(t *testing.T) {
	ledger := mock.NewMockBlockStore()
	block := mock.NewBlockBuilder().Block()
	ledger.AppendBlock(block)
	router := NewRounterFromChainMG(ledger)

	filter := &protos.BlockFilter{
		Range: &protos.BlockRange{
			Start: "0",
		},
	}

	iter, err := router.RawSubscribe(protos.SubscribeType_BLOCK, filter)
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()
	iter.Next()
	fblock := iter.Data().(*protos.FilteredBlock)

	if fblock.GetBlockid() != hex.EncodeToString(block.GetBlockid()) {
		t.Fatalf("block not equal, expect %x got %s", block.GetBlockid(), fblock.GetBlockid())
	}
}
