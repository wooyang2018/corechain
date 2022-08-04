package network

import (
	"testing"

	"github.com/wooyang2018/corechain/protos"
)

func TestMessage(t *testing.T) {
	data := &protos.CoreMessage{
		Data: &protos.CoreMessage_MessageData{
			MsgInfo: []byte("hello world"),
		},
	}

	cases := []*protos.CoreMessage{
		NewMessage(protos.CoreMessage_GET_BLOCK, data),
		NewMessage(protos.CoreMessage_GET_BLOCKCHAINSTATUS, data),
	}

	for i, req := range cases {
		var data protos.CoreMessage
		err := Unmarshal(req, &data)
		if err != nil {
			t.Errorf("case[%d]: unmarshal message error: %v", i, err)
			continue
		}

		respType := GetRespMessageType(req.GetHeader().GetType())
		resp := NewMessage(respType, &data)
		if VerifyMessageType(req, resp, "") {
			t.Errorf("case[%d]: verify message type error", i)
			continue
		}
	}

}
