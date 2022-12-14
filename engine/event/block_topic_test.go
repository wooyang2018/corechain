package event

import (
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/wooyang2018/corechain/engine/mock"
	"github.com/wooyang2018/corechain/protos"
)

func TestBlockTopicBasic(t *testing.T) {
	ledger := mock.NewMockBlockStore()
	const N = 5
	var blocks []string
	for i := 0; i < N; i++ {
		block := mock.NewBlockBuilder().Block()
		blocks = append(blocks, hex.EncodeToString(block.GetBlockid()))
		ledger.AppendBlock(block)
	}

	topic := NewBlockTopic(ledger)
	iter, err := topic.NewFilterIterator(&protos.BlockFilter{
		Range: &protos.BlockRange{
			Start: "0",
			End:   strconv.Itoa(N),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	i := 0
	for i = 0; iter.Next(); i++ {
		if i >= len(blocks) {
			t.Fatal("unexpected block event length")
		}
		block := iter.Data().(*protos.FilteredBlock)
		if block.GetBlockid() != blocks[i] {
			t.Errorf("expect %s got %s", blocks[i], block.GetBlockid())
		}
	}

	if i < len(blocks)-1 {
		t.Errorf("unexpect block event length %d", i)
	}
}

func TestBlockTopicWaitBlock(t *testing.T) {
	ledger := mock.NewMockBlockStore()
	const N = 5
	go func() {
		for i := 0; i < N; i++ {
			time.Sleep(time.Millisecond * 100)
			block := mock.NewBlockBuilder().Block()
			ledger.AppendBlock(block)
		}
	}()

	topic := NewBlockTopic(ledger)
	iter, err := topic.NewFilterIterator(&protos.BlockFilter{
		Range: &protos.BlockRange{
			Start: "0",
			End:   strconv.Itoa(N),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	for i := 0; iter.Next(); i++ {
		block := iter.Data().(*protos.FilteredBlock)
		_, err := ledger.QueryBlockByHeight(block.GetBlockHeight())
		if err != nil {
			t.Errorf("unexpect error %s", err)
		}
	}
}

func TestBlockTopicEmptyRange(t *testing.T) {
	ledger := mock.NewMockBlockStore()
	const N = 5
	go func() {
		for i := 0; i < N; i++ {
			time.Sleep(time.Millisecond * 100)
			block := mock.NewBlockBuilder().Block()
			ledger.AppendBlock(block)
		}
	}()

	topic := NewBlockTopic(ledger)
	iter, err := topic.NewFilterIterator(&protos.BlockFilter{
		Range: &protos.BlockRange{},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer iter.Close()

	for i := 0; iter.Next(); i++ {
		block := iter.Data().(*protos.FilteredBlock)
		_, err := ledger.QueryBlockByHeight(block.GetBlockHeight())
		if err != nil {
			t.Errorf("unexpect error %s", err)
		}
		if block.GetBlockHeight() >= N-1 {
			break
		}
	}
}

func TestFilterTxEvent(t *testing.T) {
	ledger := mock.NewMockBlockStore()
	tx := mock.NewTxBuilder().Invoke("counter", "increase", &protos.ContractEvent{
		Contract: "counter",
		Name:     "increase",
	}).Tx()

	block := mock.NewBlockBuilder().AddTx(tx).Block()
	ledger.AppendBlock(block)

	topic := NewBlockTopic(ledger)

	// Tx should not matched even contract name equals
	t.Run("eventNotMatch", func(tt *testing.T) {
		iter, err := topic.NewFilterIterator(&protos.BlockFilter{
			Range: &protos.BlockRange{
				Start: "0",
			},
			Contract:  "counter",
			EventName: "get",
		})
		if err != nil {
			tt.Fatal(err)
		}
		defer iter.Close()
		iter.Next()
		fblock := iter.Data().(*protos.FilteredBlock)
		if len(fblock.Txs) != 0 {
			tt.Fatal("expect empty ex")
		}
	})

	t.Run("eventMatch", func(tt *testing.T) {
		iter, err := topic.NewFilterIterator(&protos.BlockFilter{
			Range: &protos.BlockRange{
				Start: "0",
			},
			EventName: "increase",
		})
		if err != nil {
			tt.Fatal(err)
		}
		defer iter.Close()
		iter.Next()
		fblock := iter.Data().(*protos.FilteredBlock)
		if len(fblock.Txs) == 0 {
			tt.Fatal("empty ex")
		}
		if len(fblock.Txs[0].Events) == 0 {
			tt.Fatal("empty events")
		}
	})
}
