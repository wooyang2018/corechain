package model

import (
	"math/big"
	"testing"

	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

func TestMarshalMessages(t *testing.T) {
	msgs := []*protos.TxInput{
		{
			RefTxid:   []byte("tx1"),
			RefOffset: 1,
			FromAddr:  []byte("fromAddr"),
			Amount:    big.NewInt(10).Bytes(),
		},
		{
			RefTxid:   []byte("tx2"),
			RefOffset: 2,
			FromAddr:  []byte("fromAddr"),
			Amount:    big.NewInt(10).Bytes(),
		},
	}

	buf, err := MarshalMessages(msgs)
	if err != nil {
		t.Fatal(err)
	}

	var out []*protos.TxInput
	err = UnmsarshalMessages(buf, &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(msgs) {
		t.Fatalf("len not equal %d:%d", len(out), len(msgs))
	}
	for i := range msgs {
		if !proto.Equal(msgs[i], out[i]) {
			t.Fatalf("msg not equal %#v\n%#v", msgs[i], out[i])
		}
	}
}

func TestNilMessages(t *testing.T) {
	var msgs []*protos.TxInput
	var out []*protos.TxInput
	buf, err := MarshalMessages(msgs)
	if err != nil {
		t.Fatal(err)
	}
	err = UnmsarshalMessages(buf, &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("unexpected length:%d", len(out))
	}
}
