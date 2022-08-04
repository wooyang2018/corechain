package p2pv2

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/wooyang2018/corechain/network"
	"github.com/wooyang2018/corechain/protos"
)

func TestDelimitedWriter(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := NewDelimitedWriter(buf)
	reader := NewDelimitedReader(buf, 1024)
	err := writer.WriteMsg(network.NewMessage(protos.CoreMessage_GET_BLOCK, nil))
	if err != nil {
		t.Fatal(err)
	}
	err = writer.WriteMsg(network.NewMessage(protos.CoreMessage_POSTTX, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	msg := new(protos.CoreMessage)
	if err := reader.ReadMsg(msg); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n", msg)
	firstRead := reflect.ValueOf(reader).Elem().FieldByName("buf").Pointer()
	if err := reader.ReadMsg(msg); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n", msg)
	secondRead := reflect.ValueOf(reader).Elem().FieldByName("buf").Pointer()

	if firstRead != secondRead {
		t.Fatalf("reader buf byte slice pointer did not stay the same after second same size read (%d != %d).", firstRead, secondRead)
	}
}
