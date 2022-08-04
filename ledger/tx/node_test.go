package tx

import (
	"fmt"
	"testing"

	"github.com/gammazero/deque"
	"github.com/wooyang2018/corechain/protos"
)

func TestDeque(t *testing.T) {
	var q deque.Deque
	q.PushBack("foo")
	q.PushBack("bar")
	q.PushBack("baz")

	fmt.Println(q.Len())   // Prints: 3
	fmt.Println(q.Front()) // Prints: foo
	fmt.Println(q.Back())  // Prints: baz

	q.PopFront() // remove "foo"
	q.PopBack()  // remove "baz"

	q.PushFront("hello")
	q.PushBack("world")

	// Consume deque and print elements.
	for q.Len() != 0 {
		fmt.Println(q.PopFront())
	}
}
func TestNewNode(t *testing.T) {
	testTx := &protos.Transaction{
		TxInputs: []*protos.TxInput{
			{RefTxid: []byte("test")},
		},
		TxInputsExt: []*protos.TxInputExt{
			{RefTxid: []byte("test")},
		},
	}

	n := NewNode("test", testTx)
	if n.txid != "test" {
		t.Error("new node test failed")
	}

	if len(n.txInputs) != 1 {
		t.Error("new node test failed")
	}
}

func TestGetAllChildren(t *testing.T) {

}
