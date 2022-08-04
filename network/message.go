package network

import (
	"errors"
	"hash/crc32"

	"github.com/golang/snappy"
	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

var (
	ErrMessageChecksum   = errors.New("verify checksum error")
	ErrMessageDecompress = errors.New("decompress error")
	ErrMessageUnmarshal  = errors.New("message unmarshal error")
)

// NewMessage create P2P message instance with given params
func NewMessage(typ protos.CoreMessage_MessageType, message proto.Message, opts ...MessageOption) *protos.CoreMessage {
	msg := &protos.CoreMessage{
		Header: &protos.CoreMessage_MessageHeader{
			Version:        MessageVersion,
			Bcname:         BlockChain,
			Logid:          utils.GenLogId(),
			Type:           typ,
			EnableCompress: false,
			ErrorType:      protos.CoreMessage_NONE,
		},
		Data: &protos.CoreMessage_MessageData{},
	}

	if message != nil {
		data, _ := proto.Marshal(message)
		msg.Data.MsgInfo = data
	}

	for _, op := range opts {
		op(msg)
	}

	Compress(msg)
	msg.Header.DataCheckSum = Checksum(msg)
	return msg
}

// Unmarshal unmarshal msgInfo
func Unmarshal(msg *protos.CoreMessage, message proto.Message) error {
	if !VerifyChecksum(msg) {
		return ErrMessageChecksum
	}

	data, err := Decompress(msg)
	if err != nil {
		return ErrMessageDecompress
	}

	err = proto.Unmarshal(data, message)
	if err != nil {
		return ErrMessageUnmarshal
	}

	return nil
}

type MessageOption func(*protos.CoreMessage)

func WithBCName(bcname string) MessageOption {
	return func(msg *protos.CoreMessage) {
		msg.Header.Bcname = bcname
	}
}

// WithLogId set message logId
func WithLogId(logid string) MessageOption {
	return func(msg *protos.CoreMessage) {
		msg.Header.Logid = logid
	}
}

func WithVersion(version string) MessageOption {
	return func(msg *protos.CoreMessage) {
		msg.Header.Version = version
	}
}

func WithErrorType(errorType protos.CoreMessage_ErrorType) MessageOption {
	return func(msg *protos.CoreMessage) {
		msg.Header.ErrorType = errorType
	}
}

// Checksum calculate checksum of message
func Checksum(msg *protos.CoreMessage) uint32 {
	return crc32.ChecksumIEEE(msg.GetData().GetMsgInfo())
}

// VerifyChecksum verify the checksum of message
func VerifyChecksum(msg *protos.CoreMessage) bool {
	return crc32.ChecksumIEEE(msg.GetData().GetMsgInfo()) == msg.GetHeader().GetDataCheckSum()
}

// Compressed compress msg
func Compress(msg *protos.CoreMessage) *protos.CoreMessage {
	if len(msg.GetData().GetMsgInfo()) == 0 {
		return msg
	}

	if msg == nil || msg.GetHeader().GetEnableCompress() {
		return msg
	}
	msg.Data.MsgInfo = snappy.Encode(nil, msg.Data.MsgInfo)
	msg.Header.EnableCompress = true
	return msg
}

// Decompress decompress msg
func Decompress(msg *protos.CoreMessage) ([]byte, error) {
	if msg == nil || msg.Header == nil || msg.Data == nil || msg.Data.MsgInfo == nil {
		return []byte{}, errors.New("param error")
	}

	if !msg.Header.GetEnableCompress() {
		return msg.Data.MsgInfo, nil
	}

	return snappy.Decode(nil, msg.Data.MsgInfo)
}

// VerifyMessageType 用于带返回的请求场景下验证收到的消息是否为预期的消息
func VerifyMessageType(request *protos.CoreMessage, response *protos.CoreMessage, peerID string) bool {
	if response.GetHeader().GetFrom() != peerID {
		return false
	}
	if request.GetHeader().GetLogid() != response.GetHeader().GetLogid() {
		return false
	}
	if GetRespMessageType(request.GetHeader().GetType()) != response.GetHeader().GetType() {
		return false
	}

	return true
}

// 消息类型映射
// 避免每次添加新消息都要修改，制定映射关系：request = n(偶数) => response = n+1(奇数)
var requestToResponse = map[protos.CoreMessage_MessageType]protos.CoreMessage_MessageType{
	protos.CoreMessage_GET_BLOCK:                protos.CoreMessage_GET_BLOCK_RES,
	protos.CoreMessage_GET_BLOCKCHAINSTATUS:     protos.CoreMessage_GET_BLOCKCHAINSTATUS_RES,
	protos.CoreMessage_CONFIRM_BLOCKCHAINSTATUS: protos.CoreMessage_CONFIRM_BLOCKCHAINSTATUS_RES,
	protos.CoreMessage_GET_RPC_PORT:             protos.CoreMessage_GET_RPC_PORT_RES,
	protos.CoreMessage_GET_AUTHENTICATION:       protos.CoreMessage_GET_AUTHENTICATION_RES,
	protos.CoreMessage_GET_BLOCK_HEADERS:        protos.CoreMessage_GET_BLOCKS_HEADERS_RES,
	protos.CoreMessage_GET_BLOCK_TXS:            protos.CoreMessage_GET_BLOCKS_TXS_RES,
}

// GetRespMessageType get the message type
func GetRespMessageType(msgType protos.CoreMessage_MessageType) protos.CoreMessage_MessageType {
	if resp, ok := requestToResponse[msgType]; ok {
		return resp
	}

	return msgType + 1
}
