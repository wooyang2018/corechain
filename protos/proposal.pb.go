// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.21.1
// source: proposal.proto

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

type ProposalStatus int32

const (
	ProposalStatus_VOTING   ProposalStatus = 0
	ProposalStatus_SUCCESS  ProposalStatus = 1
	ProposalStatus_FAILURE  ProposalStatus = 2
	ProposalStatus_CANCELED ProposalStatus = 3
)

// Enum value maps for ProposalStatus.
var (
	ProposalStatus_name = map[int32]string{
		0: "VOTING",
		1: "SUCCESS",
		2: "FAILURE",
		3: "CANCELED",
	}
	ProposalStatus_value = map[string]int32{
		"VOTING":   0,
		"SUCCESS":  1,
		"FAILURE":  2,
		"CANCELED": 3,
	}
)

func (x ProposalStatus) Enum() *ProposalStatus {
	p := new(ProposalStatus)
	*p = x
	return p
}

func (x ProposalStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ProposalStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_proposal_proto_enumTypes[0].Descriptor()
}

func (ProposalStatus) Type() protoreflect.EnumType {
	return &file_proposal_proto_enumTypes[0]
}

func (x ProposalStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ProposalStatus.Descriptor instead.
func (ProposalStatus) EnumDescriptor() ([]byte, []int) {
	return file_proposal_proto_rawDescGZIP(), []int{0}
}

// GovernTokenBalance
type GovernTokenBalance struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TotalBalance                string `protobuf:"bytes,1,opt,name=total_balance,json=totalBalance,proto3" json:"total_balance,omitempty"`
	AvailableBalanceForTdpos    string `protobuf:"bytes,2,opt,name=available_balance_for_tdpos,json=availableBalanceForTdpos,proto3" json:"available_balance_for_tdpos,omitempty"`
	LockedBalanceForTdpos       string `protobuf:"bytes,3,opt,name=locked_balance_for_tdpos,json=lockedBalanceForTdpos,proto3" json:"locked_balance_for_tdpos,omitempty"`
	AvailableBalanceForProposal string `protobuf:"bytes,4,opt,name=available_balance_for_proposal,json=availableBalanceForProposal,proto3" json:"available_balance_for_proposal,omitempty"`
	LockedBalanceForProposal    string `protobuf:"bytes,5,opt,name=locked_balance_for_proposal,json=lockedBalanceForProposal,proto3" json:"locked_balance_for_proposal,omitempty"`
}

func (x *GovernTokenBalance) Reset() {
	*x = GovernTokenBalance{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proposal_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GovernTokenBalance) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GovernTokenBalance) ProtoMessage() {}

func (x *GovernTokenBalance) ProtoReflect() protoreflect.Message {
	mi := &file_proposal_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GovernTokenBalance.ProtoReflect.Descriptor instead.
func (*GovernTokenBalance) Descriptor() ([]byte, []int) {
	return file_proposal_proto_rawDescGZIP(), []int{0}
}

func (x *GovernTokenBalance) GetTotalBalance() string {
	if x != nil {
		return x.TotalBalance
	}
	return ""
}

func (x *GovernTokenBalance) GetAvailableBalanceForTdpos() string {
	if x != nil {
		return x.AvailableBalanceForTdpos
	}
	return ""
}

func (x *GovernTokenBalance) GetLockedBalanceForTdpos() string {
	if x != nil {
		return x.LockedBalanceForTdpos
	}
	return ""
}

func (x *GovernTokenBalance) GetAvailableBalanceForProposal() string {
	if x != nil {
		return x.AvailableBalanceForProposal
	}
	return ""
}

func (x *GovernTokenBalance) GetLockedBalanceForProposal() string {
	if x != nil {
		return x.LockedBalanceForProposal
	}
	return ""
}

// TriggerDesc
type TriggerDesc struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height int64             `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	Module string            `protobuf:"bytes,2,opt,name=module,proto3" json:"module,omitempty"`
	Method string            `protobuf:"bytes,3,opt,name=method,proto3" json:"method,omitempty"`
	Args   map[string][]byte `protobuf:"bytes,4,rep,name=args,proto3" json:"args,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *TriggerDesc) Reset() {
	*x = TriggerDesc{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proposal_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TriggerDesc) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TriggerDesc) ProtoMessage() {}

func (x *TriggerDesc) ProtoReflect() protoreflect.Message {
	mi := &file_proposal_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TriggerDesc.ProtoReflect.Descriptor instead.
func (*TriggerDesc) Descriptor() ([]byte, []int) {
	return file_proposal_proto_rawDescGZIP(), []int{1}
}

func (x *TriggerDesc) GetHeight() int64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *TriggerDesc) GetModule() string {
	if x != nil {
		return x.Module
	}
	return ""
}

func (x *TriggerDesc) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *TriggerDesc) GetArgs() map[string][]byte {
	if x != nil {
		return x.Args
	}
	return nil
}

// Proposal
type Proposal struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Module     string            `protobuf:"bytes,1,opt,name=module,proto3" json:"module,omitempty"`
	Method     string            `protobuf:"bytes,2,opt,name=method,proto3" json:"method,omitempty"`
	Args       map[string][]byte `protobuf:"bytes,3,rep,name=args,proto3" json:"args,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Trigger    *TriggerDesc      `protobuf:"bytes,4,opt,name=trigger,proto3" json:"trigger,omitempty"`
	VoteAmount string            `protobuf:"bytes,5,opt,name=vote_amount,json=voteAmount,proto3" json:"vote_amount,omitempty"`
	Status     ProposalStatus    `protobuf:"varint,6,opt,name=status,proto3,enum=protos.ProposalStatus" json:"status,omitempty"`
	Proposer   string            `protobuf:"bytes,7,opt,name=proposer,proto3" json:"proposer,omitempty"`
}

func (x *Proposal) Reset() {
	*x = Proposal{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proposal_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Proposal) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Proposal) ProtoMessage() {}

func (x *Proposal) ProtoReflect() protoreflect.Message {
	mi := &file_proposal_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Proposal.ProtoReflect.Descriptor instead.
func (*Proposal) Descriptor() ([]byte, []int) {
	return file_proposal_proto_rawDescGZIP(), []int{2}
}

func (x *Proposal) GetModule() string {
	if x != nil {
		return x.Module
	}
	return ""
}

func (x *Proposal) GetMethod() string {
	if x != nil {
		return x.Method
	}
	return ""
}

func (x *Proposal) GetArgs() map[string][]byte {
	if x != nil {
		return x.Args
	}
	return nil
}

func (x *Proposal) GetTrigger() *TriggerDesc {
	if x != nil {
		return x.Trigger
	}
	return nil
}

func (x *Proposal) GetVoteAmount() string {
	if x != nil {
		return x.VoteAmount
	}
	return ""
}

func (x *Proposal) GetStatus() ProposalStatus {
	if x != nil {
		return x.Status
	}
	return ProposalStatus_VOTING
}

func (x *Proposal) GetProposer() string {
	if x != nil {
		return x.Proposer
	}
	return ""
}

var File_proposal_proto protoreflect.FileDescriptor

var file_proposal_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x22, 0xb5, 0x02, 0x0a, 0x12, 0x47, 0x6f, 0x76,
	0x65, 0x72, 0x6e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x12,
	0x23, 0x0a, 0x0d, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x5f, 0x62, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x74, 0x6f, 0x74, 0x61, 0x6c, 0x42, 0x61, 0x6c,
	0x61, 0x6e, 0x63, 0x65, 0x12, 0x3d, 0x0a, 0x1b, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c,
	0x65, 0x5f, 0x62, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x66, 0x6f, 0x72, 0x5f, 0x74, 0x64,
	0x70, 0x6f, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x18, 0x61, 0x76, 0x61, 0x69, 0x6c,
	0x61, 0x62, 0x6c, 0x65, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x46, 0x6f, 0x72, 0x54, 0x64,
	0x70, 0x6f, 0x73, 0x12, 0x37, 0x0a, 0x18, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x5f, 0x62, 0x61,
	0x6c, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x66, 0x6f, 0x72, 0x5f, 0x74, 0x64, 0x70, 0x6f, 0x73, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x15, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x42, 0x61, 0x6c,
	0x61, 0x6e, 0x63, 0x65, 0x46, 0x6f, 0x72, 0x54, 0x64, 0x70, 0x6f, 0x73, 0x12, 0x43, 0x0a, 0x1e,
	0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x62, 0x61, 0x6c, 0x61, 0x6e, 0x63,
	0x65, 0x5f, 0x66, 0x6f, 0x72, 0x5f, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x1b, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x42,
	0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x46, 0x6f, 0x72, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61,
	0x6c, 0x12, 0x3d, 0x0a, 0x1b, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x5f, 0x62, 0x61, 0x6c, 0x61,
	0x6e, 0x63, 0x65, 0x5f, 0x66, 0x6f, 0x72, 0x5f, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x18, 0x6c, 0x6f, 0x63, 0x6b, 0x65, 0x64, 0x42, 0x61,
	0x6c, 0x61, 0x6e, 0x63, 0x65, 0x46, 0x6f, 0x72, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c,
	0x22, 0xc1, 0x01, 0x0a, 0x0b, 0x54, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x44, 0x65, 0x73, 0x63,
	0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x6f, 0x64, 0x75,
	0x6c, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65,
	0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x31, 0x0a, 0x04, 0x61, 0x72, 0x67, 0x73,
	0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e,
	0x54, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x44, 0x65, 0x73, 0x63, 0x2e, 0x41, 0x72, 0x67, 0x73,
	0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x61, 0x72, 0x67, 0x73, 0x1a, 0x37, 0x0a, 0x09, 0x41,
	0x72, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x22, 0xbf, 0x02, 0x0a, 0x08, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61,
	0x6c, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x06, 0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x74,
	0x68, 0x6f, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f,
	0x64, 0x12, 0x2e, 0x0a, 0x04, 0x61, 0x72, 0x67, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61,
	0x6c, 0x2e, 0x41, 0x72, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x61, 0x72, 0x67,
	0x73, 0x12, 0x2d, 0x0a, 0x07, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x13, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x54, 0x72, 0x69, 0x67,
	0x67, 0x65, 0x72, 0x44, 0x65, 0x73, 0x63, 0x52, 0x07, 0x74, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72,
	0x12, 0x1f, 0x0a, 0x0b, 0x76, 0x6f, 0x74, 0x65, 0x5f, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x76, 0x6f, 0x74, 0x65, 0x41, 0x6d, 0x6f, 0x75, 0x6e,
	0x74, 0x12, 0x2e, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x16, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x50, 0x72, 0x6f, 0x70, 0x6f,
	0x73, 0x61, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x08, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x72, 0x1a, 0x37, 0x0a,
	0x09, 0x41, 0x72, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65,
	0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x2a, 0x44, 0x0a, 0x0e, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73,
	0x61, 0x6c, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x0a, 0x0a, 0x06, 0x56, 0x4f, 0x54, 0x49,
	0x4e, 0x47, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x55, 0x43, 0x43, 0x45, 0x53, 0x53, 0x10,
	0x01, 0x12, 0x0b, 0x0a, 0x07, 0x46, 0x41, 0x49, 0x4c, 0x55, 0x52, 0x45, 0x10, 0x02, 0x12, 0x0c,
	0x0a, 0x08, 0x43, 0x41, 0x4e, 0x43, 0x45, 0x4c, 0x45, 0x44, 0x10, 0x03, 0x42, 0x29, 0x5a, 0x27,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x77, 0x6f, 0x6f, 0x79, 0x61,
	0x6e, 0x67, 0x32, 0x30, 0x31, 0x38, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x63, 0x68, 0x61, 0x69, 0x6e,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proposal_proto_rawDescOnce sync.Once
	file_proposal_proto_rawDescData = file_proposal_proto_rawDesc
)

func file_proposal_proto_rawDescGZIP() []byte {
	file_proposal_proto_rawDescOnce.Do(func() {
		file_proposal_proto_rawDescData = protoimpl.X.CompressGZIP(file_proposal_proto_rawDescData)
	})
	return file_proposal_proto_rawDescData
}

var file_proposal_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proposal_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_proposal_proto_goTypes = []interface{}{
	(ProposalStatus)(0),        // 0: protos.ProposalStatus
	(*GovernTokenBalance)(nil), // 1: protos.GovernTokenBalance
	(*TriggerDesc)(nil),        // 2: protos.TriggerDesc
	(*Proposal)(nil),           // 3: protos.Proposal
	nil,                        // 4: protos.TriggerDesc.ArgsEntry
	nil,                        // 5: protos.Proposal.ArgsEntry
}
var file_proposal_proto_depIdxs = []int32{
	4, // 0: protos.TriggerDesc.args:type_name -> protos.TriggerDesc.ArgsEntry
	5, // 1: protos.Proposal.args:type_name -> protos.Proposal.ArgsEntry
	2, // 2: protos.Proposal.trigger:type_name -> protos.TriggerDesc
	0, // 3: protos.Proposal.status:type_name -> protos.ProposalStatus
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_proposal_proto_init() }
func file_proposal_proto_init() {
	if File_proposal_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proposal_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GovernTokenBalance); i {
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
		file_proposal_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TriggerDesc); i {
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
		file_proposal_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Proposal); i {
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
			RawDescriptor: file_proposal_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proposal_proto_goTypes,
		DependencyIndexes: file_proposal_proto_depIdxs,
		EnumInfos:         file_proposal_proto_enumTypes,
		MessageInfos:      file_proposal_proto_msgTypes,
	}.Build()
	File_proposal_proto = out.File
	file_proposal_proto_rawDesc = nil
	file_proposal_proto_goTypes = nil
	file_proposal_proto_depIdxs = nil
}