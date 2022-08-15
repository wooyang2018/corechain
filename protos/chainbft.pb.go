// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.21.1
// source: chainbft.proto

package protos

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// QCState is the phase of hotstuff
type QCState int32

const (
	QCState_NEW_VIEW   QCState = 0
	QCState_PREPARE    QCState = 1
	QCState_PRE_COMMIT QCState = 2
	QCState_COMMIT     QCState = 3
	QCState_DECIDE     QCState = 4
)

// Enum value maps for QCState.
var (
	QCState_name = map[int32]string{
		0: "NEW_VIEW",
		1: "PREPARE",
		2: "PRE_COMMIT",
		3: "COMMIT",
		4: "DECIDE",
	}
	QCState_value = map[string]int32{
		"NEW_VIEW":   0,
		"PREPARE":    1,
		"PRE_COMMIT": 2,
		"COMMIT":     3,
		"DECIDE":     4,
	}
)

func (x QCState) Enum() *QCState {
	p := new(QCState)
	*p = x
	return p
}

func (x QCState) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (QCState) Descriptor() protoreflect.EnumDescriptor {
	return file_chainbft_proto_enumTypes[0].Descriptor()
}

func (QCState) Type() protoreflect.EnumType {
	return &file_chainbft_proto_enumTypes[0]
}

func (x QCState) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use QCState.Descriptor instead.
func (QCState) EnumDescriptor() ([]byte, []int) {
	return file_chainbft_proto_rawDescGZIP(), []int{0}
}

// QuorumCert is a data type that combines a collection of signatures from replicas.
type QuorumCert struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The id of Proposal this QC certified.
	ProposalId []byte `protobuf:"bytes,1,opt,name=ProposalId,proto3" json:"ProposalId,omitempty"`
	// The msg of Proposal this QC certified.
	ProposalMsg []byte `protobuf:"bytes,2,opt,name=ProposalMsg,proto3" json:"ProposalMsg,omitempty"`
	// The current type of this QC certified.
	// the type contains `NEW_VIEW`, `PREPARE`
	Type QCState `protobuf:"varint,3,opt,name=Type,proto3,enum=protos.QCState" json:"Type,omitempty"`
	// The view number of this QC certified.
	ViewNumber int64 `protobuf:"varint,4,opt,name=ViewNumber,proto3" json:"ViewNumber,omitempty"`
	// SignInfos is the signs of the leader gathered from replicas
	// of a specifically certType.
	SignInfos *QCSignInfos `protobuf:"bytes,5,opt,name=SignInfos,proto3" json:"SignInfos,omitempty"`
}

func (x *QuorumCert) Reset() {
	*x = QuorumCert{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chainbft_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuorumCert) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuorumCert) ProtoMessage() {}

func (x *QuorumCert) ProtoReflect() protoreflect.Message {
	mi := &file_chainbft_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuorumCert.ProtoReflect.Descriptor instead.
func (*QuorumCert) Descriptor() ([]byte, []int) {
	return file_chainbft_proto_rawDescGZIP(), []int{0}
}

func (x *QuorumCert) GetProposalId() []byte {
	if x != nil {
		return x.ProposalId
	}
	return nil
}

func (x *QuorumCert) GetProposalMsg() []byte {
	if x != nil {
		return x.ProposalMsg
	}
	return nil
}

func (x *QuorumCert) GetType() QCState {
	if x != nil {
		return x.Type
	}
	return QCState_NEW_VIEW
}

func (x *QuorumCert) GetViewNumber() int64 {
	if x != nil {
		return x.ViewNumber
	}
	return 0
}

func (x *QuorumCert) GetSignInfos() *QCSignInfos {
	if x != nil {
		return x.SignInfos
	}
	return nil
}

// QCSignInfos is the signs of the leader gathered from replicas of a specifically certType.
// A slice of signs is used at present.
// TODO @qizheng09: It will be change to Threshold-Signatures after
// Crypto lib support Threshold-Signatures.
type QCSignInfos struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// QCSignInfos
	QCSignInfos []*SignInfo `protobuf:"bytes,1,rep,name=QCSignInfos,proto3" json:"QCSignInfos,omitempty"`
}

func (x *QCSignInfos) Reset() {
	*x = QCSignInfos{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chainbft_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QCSignInfos) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QCSignInfos) ProtoMessage() {}

func (x *QCSignInfos) ProtoReflect() protoreflect.Message {
	mi := &file_chainbft_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QCSignInfos.ProtoReflect.Descriptor instead.
func (*QCSignInfos) Descriptor() ([]byte, []int) {
	return file_chainbft_proto_rawDescGZIP(), []int{1}
}

func (x *QCSignInfos) GetQCSignInfos() []*SignInfo {
	if x != nil {
		return x.QCSignInfos
	}
	return nil
}

// SignInfo is the signature information of the
type SignInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address   string `protobuf:"bytes,1,opt,name=Address,proto3" json:"Address,omitempty"`
	PublicKey string `protobuf:"bytes,2,opt,name=PublicKey,proto3" json:"PublicKey,omitempty"`
	Sign      []byte `protobuf:"bytes,3,opt,name=Sign,proto3" json:"Sign,omitempty"`
}

func (x *SignInfo) Reset() {
	*x = SignInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chainbft_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SignInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SignInfo) ProtoMessage() {}

func (x *SignInfo) ProtoReflect() protoreflect.Message {
	mi := &file_chainbft_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SignInfo.ProtoReflect.Descriptor instead.
func (*SignInfo) Descriptor() ([]byte, []int) {
	return file_chainbft_proto_rawDescGZIP(), []int{2}
}

func (x *SignInfo) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *SignInfo) GetPublicKey() string {
	if x != nil {
		return x.PublicKey
	}
	return ""
}

func (x *SignInfo) GetSign() []byte {
	if x != nil {
		return x.Sign
	}
	return nil
}

// QuorumCertSign 是(Addr, Pk, 签名)三元组
type QuorumCertSign struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address   string `protobuf:"bytes,1,opt,name=Address,proto3" json:"Address,omitempty"`
	PublicKey string `protobuf:"bytes,2,opt,name=PublicKey,proto3" json:"PublicKey,omitempty"`
	Sign      []byte `protobuf:"bytes,3,opt,name=Sign,proto3" json:"Sign,omitempty"`
}

func (x *QuorumCertSign) Reset() {
	*x = QuorumCertSign{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chainbft_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *QuorumCertSign) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*QuorumCertSign) ProtoMessage() {}

func (x *QuorumCertSign) ProtoReflect() protoreflect.Message {
	mi := &file_chainbft_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use QuorumCertSign.ProtoReflect.Descriptor instead.
func (*QuorumCertSign) Descriptor() ([]byte, []int) {
	return file_chainbft_proto_rawDescGZIP(), []int{3}
}

func (x *QuorumCertSign) GetAddress() string {
	if x != nil {
		return x.Address
	}
	return ""
}

func (x *QuorumCertSign) GetPublicKey() string {
	if x != nil {
		return x.PublicKey
	}
	return ""
}

func (x *QuorumCertSign) GetSign() []byte {
	if x != nil {
		return x.Sign
	}
	return nil
}

// ProposalMsg 是chained-bft中定义的Block形式，区别在于其有一个parentQC，该存储只供chained-bft类使用
type ProposalMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// 生产高度
	ProposalView int64  `protobuf:"varint,1,opt,name=proposalView,proto3" json:"proposalView,omitempty"`
	ProposalId   []byte `protobuf:"bytes,2,opt,name=proposalId,proto3" json:"proposalId,omitempty"`
	// 生产时间
	Timestamp int64 `protobuf:"varint,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	// 上一个区块基本信息
	JustifyQC []byte `protobuf:"bytes,4,opt,name=JustifyQC,proto3" json:"JustifyQC,omitempty"`
	// 签名
	Sign *QuorumCertSign `protobuf:"bytes,5,opt,name=Sign,proto3" json:"Sign,omitempty"`
	// 消息摘要
	MsgDigest []byte `protobuf:"bytes,6,opt,name=MsgDigest,proto3" json:"MsgDigest,omitempty"`
}

func (x *ProposalMsg) Reset() {
	*x = ProposalMsg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chainbft_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProposalMsg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProposalMsg) ProtoMessage() {}

func (x *ProposalMsg) ProtoReflect() protoreflect.Message {
	mi := &file_chainbft_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProposalMsg.ProtoReflect.Descriptor instead.
func (*ProposalMsg) Descriptor() ([]byte, []int) {
	return file_chainbft_proto_rawDescGZIP(), []int{4}
}

func (x *ProposalMsg) GetProposalView() int64 {
	if x != nil {
		return x.ProposalView
	}
	return 0
}

func (x *ProposalMsg) GetProposalId() []byte {
	if x != nil {
		return x.ProposalId
	}
	return nil
}

func (x *ProposalMsg) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *ProposalMsg) GetJustifyQC() []byte {
	if x != nil {
		return x.JustifyQC
	}
	return nil
}

func (x *ProposalMsg) GetSign() *QuorumCertSign {
	if x != nil {
		return x.Sign
	}
	return nil
}

func (x *ProposalMsg) GetMsgDigest() []byte {
	if x != nil {
		return x.MsgDigest
	}
	return nil
}

// VoteMsg is the vote message of the protocal.
type VoteMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	VoteInfo         []byte            `protobuf:"bytes,1,opt,name=VoteInfo,proto3" json:"VoteInfo,omitempty"`
	LedgerCommitInfo []byte            `protobuf:"bytes,2,opt,name=LedgerCommitInfo,proto3" json:"LedgerCommitInfo,omitempty"`
	Signature        []*QuorumCertSign `protobuf:"bytes,3,rep,name=Signature,proto3" json:"Signature,omitempty"`
}

func (x *VoteMsg) Reset() {
	*x = VoteMsg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chainbft_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *VoteMsg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*VoteMsg) ProtoMessage() {}

func (x *VoteMsg) ProtoReflect() protoreflect.Message {
	mi := &file_chainbft_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use VoteMsg.ProtoReflect.Descriptor instead.
func (*VoteMsg) Descriptor() ([]byte, []int) {
	return file_chainbft_proto_rawDescGZIP(), []int{5}
}

func (x *VoteMsg) GetVoteInfo() []byte {
	if x != nil {
		return x.VoteInfo
	}
	return nil
}

func (x *VoteMsg) GetLedgerCommitInfo() []byte {
	if x != nil {
		return x.LedgerCommitInfo
	}
	return nil
}

func (x *VoteMsg) GetSignature() []*QuorumCertSign {
	if x != nil {
		return x.Signature
	}
	return nil
}

var File_chainbft_proto protoreflect.FileDescriptor

var file_chainbft_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x62, 0x66, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x22, 0xc6, 0x01, 0x0a, 0x0a, 0x51, 0x75, 0x6f,
	0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x12, 0x1e, 0x0a, 0x0a, 0x50, 0x72, 0x6f, 0x70, 0x6f,
	0x73, 0x61, 0x6c, 0x49, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x50, 0x72, 0x6f,
	0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x64, 0x12, 0x20, 0x0a, 0x0b, 0x50, 0x72, 0x6f, 0x70, 0x6f,
	0x73, 0x61, 0x6c, 0x4d, 0x73, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0b, 0x50, 0x72,
	0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x4d, 0x73, 0x67, 0x12, 0x23, 0x0a, 0x04, 0x54, 0x79, 0x70,
	0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73,
	0x2e, 0x51, 0x43, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x1e,
	0x0a, 0x0a, 0x56, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x0a, 0x56, 0x69, 0x65, 0x77, 0x4e, 0x75, 0x6d, 0x62, 0x65, 0x72, 0x12, 0x31,
	0x0a, 0x09, 0x53, 0x69, 0x67, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x13, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x51, 0x43, 0x53, 0x69, 0x67,
	0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x73, 0x52, 0x09, 0x53, 0x69, 0x67, 0x6e, 0x49, 0x6e, 0x66, 0x6f,
	0x73, 0x22, 0x41, 0x0a, 0x0b, 0x51, 0x43, 0x53, 0x69, 0x67, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x73,
	0x12, 0x32, 0x0a, 0x0b, 0x51, 0x43, 0x53, 0x69, 0x67, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x53,
	0x69, 0x67, 0x6e, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x0b, 0x51, 0x43, 0x53, 0x69, 0x67, 0x6e, 0x49,
	0x6e, 0x66, 0x6f, 0x73, 0x22, 0x56, 0x0a, 0x08, 0x53, 0x69, 0x67, 0x6e, 0x49, 0x6e, 0x66, 0x6f,
	0x12, 0x18, 0x0a, 0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x50, 0x75,
	0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x50,
	0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x53, 0x69, 0x67, 0x6e,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x53, 0x69, 0x67, 0x6e, 0x22, 0x5c, 0x0a, 0x0e,
	0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x53, 0x69, 0x67, 0x6e, 0x12, 0x18,
	0x0a, 0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x1c, 0x0a, 0x09, 0x50, 0x75, 0x62, 0x6c,
	0x69, 0x63, 0x4b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x50, 0x75, 0x62,
	0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x53, 0x69, 0x67, 0x6e, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x53, 0x69, 0x67, 0x6e, 0x22, 0xd7, 0x01, 0x0a, 0x0b, 0x50,
	0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x4d, 0x73, 0x67, 0x12, 0x22, 0x0a, 0x0c, 0x70, 0x72,
	0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x56, 0x69, 0x65, 0x77, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x0c, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x56, 0x69, 0x65, 0x77, 0x12, 0x1e,
	0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x0a, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x64, 0x12, 0x1c,
	0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x1c, 0x0a, 0x09,
	0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x79, 0x51, 0x43, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x09, 0x4a, 0x75, 0x73, 0x74, 0x69, 0x66, 0x79, 0x51, 0x43, 0x12, 0x2a, 0x0a, 0x04, 0x53, 0x69,
	0x67, 0x6e, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x73, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x53, 0x69, 0x67, 0x6e,
	0x52, 0x04, 0x53, 0x69, 0x67, 0x6e, 0x12, 0x1c, 0x0a, 0x09, 0x4d, 0x73, 0x67, 0x44, 0x69, 0x67,
	0x65, 0x73, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x4d, 0x73, 0x67, 0x44, 0x69,
	0x67, 0x65, 0x73, 0x74, 0x22, 0x87, 0x01, 0x0a, 0x07, 0x56, 0x6f, 0x74, 0x65, 0x4d, 0x73, 0x67,
	0x12, 0x1a, 0x0a, 0x08, 0x56, 0x6f, 0x74, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x08, 0x56, 0x6f, 0x74, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x2a, 0x0a, 0x10,
	0x4c, 0x65, 0x64, 0x67, 0x65, 0x72, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e, 0x66, 0x6f,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x10, 0x4c, 0x65, 0x64, 0x67, 0x65, 0x72, 0x43, 0x6f,
	0x6d, 0x6d, 0x69, 0x74, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x34, 0x0a, 0x09, 0x53, 0x69, 0x67, 0x6e,
	0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x51, 0x75, 0x6f, 0x72, 0x75, 0x6d, 0x43, 0x65, 0x72, 0x74, 0x53,
	0x69, 0x67, 0x6e, 0x52, 0x09, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x2a, 0x4c,
	0x0a, 0x07, 0x51, 0x43, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x0c, 0x0a, 0x08, 0x4e, 0x45, 0x57,
	0x5f, 0x56, 0x49, 0x45, 0x57, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x50, 0x52, 0x45, 0x50, 0x41,
	0x52, 0x45, 0x10, 0x01, 0x12, 0x0e, 0x0a, 0x0a, 0x50, 0x52, 0x45, 0x5f, 0x43, 0x4f, 0x4d, 0x4d,
	0x49, 0x54, 0x10, 0x02, 0x12, 0x0a, 0x0a, 0x06, 0x43, 0x4f, 0x4d, 0x4d, 0x49, 0x54, 0x10, 0x03,
	0x12, 0x0a, 0x0a, 0x06, 0x44, 0x45, 0x43, 0x49, 0x44, 0x45, 0x10, 0x04, 0x42, 0x29, 0x5a, 0x27,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x77, 0x6f, 0x6f, 0x79, 0x61,
	0x6e, 0x67, 0x32, 0x30, 0x31, 0x38, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x63, 0x68, 0x61, 0x69, 0x6e,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_chainbft_proto_rawDescOnce sync.Once
	file_chainbft_proto_rawDescData = file_chainbft_proto_rawDesc
)

func file_chainbft_proto_rawDescGZIP() []byte {
	file_chainbft_proto_rawDescOnce.Do(func() {
		file_chainbft_proto_rawDescData = protoimpl.X.CompressGZIP(file_chainbft_proto_rawDescData)
	})
	return file_chainbft_proto_rawDescData
}

var file_chainbft_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_chainbft_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_chainbft_proto_goTypes = []interface{}{
	(QCState)(0),           // 0: protos.QCState
	(*QuorumCert)(nil),     // 1: protos.QuorumCert
	(*QCSignInfos)(nil),    // 2: protos.QCSignInfos
	(*SignInfo)(nil),       // 3: protos.SignInfo
	(*QuorumCertSign)(nil), // 4: protos.QuorumCertSign
	(*ProposalMsg)(nil),    // 5: protos.ProposalMsg
	(*VoteMsg)(nil),        // 6: protos.VoteMsg
}
var file_chainbft_proto_depIdxs = []int32{
	0, // 0: protos.QuorumCert.Type:type_name -> protos.QCState
	2, // 1: protos.QuorumCert.SignInfos:type_name -> protos.QCSignInfos
	3, // 2: protos.QCSignInfos.QCSignInfos:type_name -> protos.SignInfo
	4, // 3: protos.ProposalMsg.Sign:type_name -> protos.QuorumCertSign
	4, // 4: protos.VoteMsg.Signature:type_name -> protos.QuorumCertSign
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_chainbft_proto_init() }
func file_chainbft_proto_init() {
	if File_chainbft_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_chainbft_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuorumCert); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chainbft_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QCSignInfos); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chainbft_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SignInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chainbft_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*QuorumCertSign); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chainbft_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProposalMsg); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chainbft_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*VoteMsg); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_chainbft_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_chainbft_proto_goTypes,
		DependencyIndexes: file_chainbft_proto_depIdxs,
		EnumInfos:         file_chainbft_proto_enumTypes,
		MessageInfos:      file_chainbft_proto_msgTypes,
	}.Build()
	File_chainbft_proto = out.File
	file_chainbft_proto_rawDesc = nil
	file_chainbft_proto_goTypes = nil
	file_chainbft_proto_depIdxs = nil
}
