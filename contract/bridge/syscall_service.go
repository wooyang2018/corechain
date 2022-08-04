// Copyright (c) 2019, Baidu.com, Inc. All Rights Reserved.

package bridge

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/proposal/utils"
	"github.com/wooyang2018/corechain/protos"
)

var (
	// ErrOutOfDiskLimit define OutOfDiskLimit Error
	ErrOutOfDiskLimit    = errors.New("out of disk limit")
	ErrNotImplementation = errors.New("not implementation")
)

const (
	// DefaultCap define default cap of NewIterator
	DefaultCap = 1000
	// MaxContractCallDepth define max contract call depth
	MaxContractCallDepth = 10
)

// SyscallService is the handler of contract syscalls
type SyscallService struct {
	ctxmgr *ContextManager
	bridge *XBridge
	core   base.ChainCore
}

// NewSyscallService instances a new SyscallService
func NewSyscallService(ctxmgr *ContextManager, bridge *XBridge) *SyscallService {
	return &SyscallService{
		ctxmgr: ctxmgr,
		bridge: bridge,
	}
}

// Ping implements Syscall interface
func (c *SyscallService) Ping(ctx context.Context, in *protos.PingRequest) (*protos.PingResponse, error) {
	return new(protos.PingResponse), nil
}

// QueryBlock implements Syscall interface
func (c *SyscallService) QueryBlock(ctx context.Context, in *protos.QueryBlockRequest) (*protos.QueryBlockResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}

	rawBlockid, err := hex.DecodeString(in.Blockid)
	if err != nil {
		return nil, err
	}

	block, err := nctx.Core.QueryBlock(rawBlockid)
	if err != nil {
		return nil, err
	}
	blocksdk := &protos.Block{
		Blockid:   hex.EncodeToString(block.GetBlockid()),
		PreHash:   hex.EncodeToString(block.GetPreHash()),
		Proposer:  block.GetProposer(),
		Sign:      hex.EncodeToString(block.GetSign()),
		Pubkey:    []byte(block.GetPublicKey()),
		Height:    block.GetHeight(),
		Timestamp: block.GetTimestamp(),
		Txids:     block.GetTxIDs(),
		TxCount:   int32(len(block.GetTxIDs())),
		InTrunk:   block.GetInTrunk(),
		NextHash:  hex.EncodeToString(block.GetNextHash()),
	}

	return &protos.QueryBlockResponse{
		Block: blocksdk,
	}, nil
}

// QueryTx implements Syscall interface
func (c *SyscallService) QueryTx(ctx context.Context, in *protos.QueryTxRequest) (*protos.QueryTxResponse, error) {

	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}

	rawTxid, err := hex.DecodeString(in.Txid)
	if err != nil {
		return nil, err
	}

	tx, err := nctx.Core.QueryTransaction(rawTxid)
	if err != nil {
		return nil, err
	}

	return &protos.QueryTxResponse{
		Tx: tx,
	}, nil
}

// Transfer implements Syscall interface
func (c *SyscallService) Transfer(ctx context.Context, in *protos.TransferRequest) (*protos.TransferResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}
	amount, ok := new(big.Int).SetString(in.GetAmount(), 10)
	if !ok {
		return nil, errors.New("parse amount error")
	}
	// make sure amount is not negative
	if amount.Cmp(new(big.Int)) < 0 {
		return nil, errors.New("amount should not be negative")
	}
	if in.GetTo() == "" {
		return nil, errors.New("empty to address")
	}
	err := nctx.State.Transfer(nctx.ContractName, in.GetTo(), amount)
	if err != nil {
		return nil, err
	}
	resp := &protos.TransferResponse{}
	return resp, nil
}

// ContractCall implements Syscall interface
func (c *SyscallService) ContractCall(ctx context.Context, in *protos.ContractCallRequest) (*protos.ContractCallResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}
	if nctx.ContractSet[in.GetContract()] && in.GetContract() != utils.TimerTaskKernelContract {
		return nil, errors.New("recursive contract call not permitted")
	}

	if len(nctx.ContractSet) >= MaxContractCallDepth {
		return nil, errors.New("max contract call depth exceeds")
	}

	ok, err := nctx.Core.VerifyContractPermission(nctx.Initiator, nctx.AuthRequire, in.GetContract(), in.GetMethod())
	if !ok || err != nil {
		return nil, errors.New("verify contract permission failed")
	}

	currentUsed := nctx.ResourceUsed()
	limits := new(base.Limits).Add(nctx.ResourceLimits).Sub(currentUsed)
	// disk usage is shared between all context
	limits.Disk = nctx.ResourceLimits.Disk

	args := make(map[string][]byte)
	for _, arg := range in.GetArgs() {
		args[arg.GetKey()] = arg.GetValue()
	}

	nctx.ContractSet[in.GetContract()] = true
	cfg := &base.ContextConfig{
		Module:         in.GetModule(),
		ContractName:   in.GetContract(),
		State:          nctx.State,
		CanInitialize:  false,
		AuthRequire:    nctx.AuthRequire,
		Initiator:      nctx.Initiator,
		Caller:         nctx.ContractName,
		ResourceLimits: *limits,
		ContractSet:    nctx.ContractSet,
	}
	vctx, err := c.bridge.NewContext(cfg)
	if err != nil {
		return nil, err
	}
	defer func() {
		vctx.Release()
		delete(nctx.ContractSet, in.GetContract())
	}()

	vresp, err := vctx.Invoke(in.GetMethod(), args)
	if err != nil {
		return nil, err
	}
	nctx.SubResourceUsed.Add(vctx.ResourceUsed())

	return &protos.ContractCallResponse{
		Response: &protos.Response{
			Status:  int32(vresp.Status),
			Message: vresp.Message,
			Body:    vresp.Body,
		}}, nil
}

// CrossContractQuery implements Syscall interface
func (c *SyscallService) CrossContractQuery(ctx context.Context, in *protos.CrossContractQueryRequest) (*protos.CrossContractQueryResponse, error) {
	return nil, ErrNotImplementation
	// nctx, ok := c.ctxmgr.VMContext(in.GetHeader().Ctxid)
	// if !ok {
	// 	return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	// }

	// crossChainURI, err := ParseCrossChainURI(in.GetUri())
	// if err != nil {
	// 	return nil, fmt.Errorf("ParseCrossChainURI error, err:%s ctx id:%d", err.Error(), in.Header.Ctxid)
	// }

	// crossQueryMeta, err := nctx.Core.ResolveChain(crossChainURI.GetChainName())
	// if err != nil {
	// 	return nil, fmt.Errorf("ResolveChain error, err:%s ctx id:%d", err.Error(), in.Header.Ctxid)
	// }

	// // Assemble crossQueryRequest
	// crossQueryRequest := &xchainpb.CrossQueryRequest{}
	// crossScheme := GetChainScheme(crossChainURI.GetScheme())
	// crossQueryRequest, err = crossScheme.GetCrossQueryRequest(crossChainURI, in.GetArgs(), nctx.GetInitiator(), nctx.GetAuthRequire())
	// if err != nil {
	// 	return nil, fmt.Errorf("GetCrossQueryRequest error, err:%s ctx id: %d", err.Error(), in.Header.Ctxid)
	// }

	// // CrossQuery cross query from other chain
	// contractResponse, err := nctx.Store.CrossQuery(crossQueryRequest, crossQueryMeta)
	// return &pb.CrossContractQueryResponse{
	// 	Response: &pb.Response{
	// 		Status:  contractResponse.GetStatus(),
	// 		Message: contractResponse.GetMessage(),
	// 		Body:    contractResponse.GetBody(),
	// 	},
	// }, err
}

// PutObject implements Syscall interface
func (c *SyscallService) PutObject(ctx context.Context, in *protos.PutRequest) (*protos.PutResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}
	if in.Value == nil {
		return nil, errors.New("put nil value")
	}

	err := nctx.State.Put(nctx.ContractName, in.Key, in.Value)
	if err != nil {
		return nil, err
	}

	return &protos.PutResponse{}, nil
}

// GetObject implements Syscall interface
func (c *SyscallService) GetObject(ctx context.Context, in *protos.GetRequest) (*protos.GetResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}

	value, err := nctx.State.Get(nctx.ContractName, in.Key)
	if err != nil {
		return nil, err
	}
	return &protos.GetResponse{
		Value: value,
	}, nil
}

// DeleteObject implements Syscall interface
func (c *SyscallService) DeleteObject(ctx context.Context, in *protos.DeleteRequest) (*protos.DeleteResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}

	err := nctx.State.Del(nctx.ContractName, in.Key)
	if err != nil {
		return nil, err
	}
	return &protos.DeleteResponse{}, nil
}

// NewIterator implements Syscall interface
func (c *SyscallService) NewIterator(ctx context.Context, in *protos.IteratorRequest) (*protos.IteratorResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}

	limit := in.Cap
	if limit <= 0 {
		limit = DefaultCap
	}
	iter, err := nctx.State.Select(nctx.ContractName, in.Start, in.Limit)
	if err != nil {
		return nil, err
	}
	out := new(protos.IteratorResponse)
	for iter.Next() && limit > 0 {
		out.Items = append(out.Items, &protos.IteratorItem{
			Key:   append([]byte(""), iter.Key()...), //make a copy
			Value: append([]byte(""), iter.Value()...),
		})
		limit -= 1
	}
	if iter.Error() != nil {
		return nil, err
	}
	iter.Close()
	return out, nil
}

// GetCallArgs implements Syscall interface
func (c *SyscallService) GetCallArgs(ctx context.Context, in *protos.GetCallArgsRequest) (*protos.CallArgs, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}
	var args []*protos.ArgPair
	for key, value := range nctx.Args {
		args = append(args, &protos.ArgPair{
			Key:   key,
			Value: value,
		})
	}
	sort.Slice(args, func(i, j int) bool {
		return args[i].Key < args[j].Key
	})
	return &protos.CallArgs{
		Method:         nctx.Method,
		Args:           args,
		Initiator:      nctx.Initiator,
		AuthRequire:    nctx.AuthRequire,
		TransferAmount: nctx.TransferAmount,
		Caller:         nctx.Caller,
	}, nil
}

// SetOutput implements Syscall interface
func (c *SyscallService) SetOutput(ctx context.Context, in *protos.SetOutputRequest) (*protos.SetOutputResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.Header.Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}
	nctx.Output = in.GetResponse()
	return new(protos.SetOutputResponse), nil
}

// GetAccountAddresses mplements Syscall interface
func (c *SyscallService) GetAccountAddresses(ctx context.Context, in *protos.GetAccountAddressesRequest) (*protos.GetAccountAddressesResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().Ctxid)
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}
	addresses, err := nctx.Core.GetAccountAddresses(in.GetAccount())
	if err != nil {
		return nil, err
	}
	return &protos.GetAccountAddressesResponse{
		Addresses: addresses,
	}, nil
}

// PostLog handle log entry from contract
func (c *SyscallService) PostLog(ctx context.Context, in *protos.PostLogRequest) (*protos.PostLogResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().GetCtxid())
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.Header.Ctxid)
	}
	nctx.Logger.SetCommField("contract_name", nctx.ContractName)
	nctx.Logger.Info(in.GetEntry())
	return &protos.PostLogResponse{}, nil
}

// PostLog handle log entry from contract
func (c *SyscallService) EmitEvent(ctx context.Context, in *protos.EmitEventRequest) (*protos.EmitEventResponse, error) {
	nctx, ok := c.ctxmgr.Context(in.GetHeader().GetCtxid())
	if !ok {
		return nil, fmt.Errorf("bad ctx id:%d", in.GetHeader().GetCtxid())
	}
	event := &protos.ContractEvent{
		Contract: nctx.ContractName,
		Name:     in.GetName(),
		Body:     in.GetBody(),
	}
	nctx.Events = append(nctx.Events, event)
	nctx.State.AddEvent(event)
	return &protos.EmitEventResponse{}, nil
}

// ConvertTxToSDKTx mplements Syscall interface
// func ConvertTxToSDKTx(tx *xpb.Transaction) *pb.Transaction {
// 	txIns := []*pb.TxInput{}
// 	for _, in := range tx.TxInputs {
// 		txIn := &pb.TxInput{
// 			RefTxid:      hex.EncodeToString(in.RefTxid),
// 			RefOffset:    in.RefOffset,
// 			FromAddr:     in.FromAddr,
// 			Amount:       AmountBytesToString(in.Amount),
// 			FrozenHeight: in.FrozenHeight,
// 		}
// 		txIns = append(txIns, txIn)
// 	}

// 	txOuts := []*pb.TxOutput{}
// 	for _, out := range tx.TxOutputs {
// 		txOut := &pb.TxOutput{
// 			Amount:       AmountBytesToString(out.Amount),
// 			ToAddr:       out.ToAddr,
// 			FrozenHeight: out.FrozenHeight,
// 		}
// 		txOuts = append(txOuts, txOut)
// 	}

// 	txsdk := &pb.Transaction{
// 		Txid:        hex.EncodeToString(tx.Txid),
// 		Blockid:     hex.EncodeToString(tx.Blockid),
// 		TxInputs:    txIns,
// 		TxOutputs:   txOuts,
// 		Desc:        tx.Desc,
// 		Initiator:   tx.Initiator,
// 		AuthRequire: tx.AuthRequire,
// 	}

// 	return txsdk
// }

// AmountBytesToString conver amount bytes to string
func AmountBytesToString(buf []byte) string {
	n := new(big.Int)
	n.SetBytes(buf)
	return n.String()
}
