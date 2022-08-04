package rpc

import (
	"context"
	"math/big"

	"github.com/wooyang2018/corechain/common/utils"
	engineBase "github.com/wooyang2018/corechain/engines/base"
	sctx "github.com/wooyang2018/corechain/example/base"
	"github.com/wooyang2018/corechain/example/models"
	protos2 "github.com/wooyang2018/corechain/example/protos"
	scom "github.com/wooyang2018/corechain/example/service/common"
	"github.com/wooyang2018/corechain/example/service/protos"
	"github.com/wooyang2018/corechain/network"
	"github.com/wooyang2018/corechain/protos"
)

// 注意：
// 1.rpc接口响应resp不能为nil，必须实例化
// 2.rpc接口响应err必须为ecom.Error类型的标准错误，没有错误响应err=nil
// 3.rpc接口不需要关注resp.Header，由拦截器根据err统一设置
// 4.rpc接口可以调用log库提供的SetInfoField方法附加输出到ending log

// PostTx post transaction to blockchain network
func (t *RPCServ) PostTx(gctx context.Context, req *protos2.TxStatus) (*protos2.CommonReply, error) {
	// 默认响应
	resp := &protos2.CommonReply{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	// 校验参数
	if req == nil || req.GetTx() == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}
	tx := scom.TxToXledger(req.GetTx())
	if tx == nil {
		rctx.GetLog().Warn("param error,tx convert to xledger tx failed")
		return resp, engineBase.ErrParameter
	}

	// 提交交易
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}

	err = handle.SubmitTx(tx)
	if err == nil {
		msg := network.NewMessage(protos.CoreMessage_POSTTX, tx,
			network.WithBCName(req.GetBcname()),
			network.WithLogId(rctx.GetLog().GetLogId()),
		)
		go t.engine.Context().Net.SendMessage(rctx, msg)
	}
	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("txid", utils.F(req.GetTxid()))
	return resp, err
}

// PreExec smart contract preExec process
func (t *RPCServ) PreExec(gctx context.Context, req *protos2.InvokeRPCRequest) (*protos2.InvokeRPCResponse, error) {
	// 默认响应
	resp := &protos2.InvokeRPCResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	// 校验参数
	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}
	reqs, err := scom.ConvertInvokeReq(req.GetRequests())
	if err != nil {
		rctx.GetLog().Warn("param error, convert failed", "err", err)
		return resp, engineBase.ErrParameter
	}

	// 预执行
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	res, err := handle.PreExec(reqs, req.GetInitiator(), req.GetAuthRequire())
	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("initiator", req.GetInitiator())
	// 设置响应
	if err == nil {
		resp.Bcname = req.GetBcname()
		resp.Response = scom.ConvertInvokeResp(res)
	}

	return resp, err
}

// PreExecWithSelectUTXO preExec + selectUtxo
func (t *RPCServ) PreExecWithSelectUTXO(gctx context.Context,
	req *protos2.PreExecWithSelectUTXORequest) (*protos2.PreExecWithSelectUTXOResponse, error) {

	// 默认响应
	resp := &protos2.PreExecWithSelectUTXOResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetRequest() == nil {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	// PreExec
	preExecRes, err := t.PreExec(gctx, req.GetRequest())
	if err != nil {
		rctx.GetLog().Warn("pre exec failed", "err", err)
		return resp, err
	}

	// SelectUTXO
	totalAmount := req.GetTotalAmount() + preExecRes.GetResponse().GetGasUsed()
	if totalAmount < 1 {
		return resp, nil
	}
	utxoInput := &protos2.UtxoInput{
		Header:    req.GetHeader(),
		Bcname:    req.GetBcname(),
		Address:   req.GetAddress(),
		Publickey: req.GetSignInfo().GetPublicKey(),
		TotalNeed: big.NewInt(totalAmount).String(),
		UserSign:  req.GetSignInfo().GetSign(),
		NeedLock:  req.GetNeedLock(),
	}
	utxoOut, err := t.SelectUTXO(gctx, utxoInput)
	if err != nil {
		return resp, err
	}
	utxoOut.Header = req.GetHeader()

	// 设置响应
	resp.Bcname = req.GetBcname()
	resp.Response = preExecRes.GetResponse()
	resp.UtxoOutput = utxoOut

	return resp, nil
}

// SelectUTXO select utxo inputs depending on amount
func (t *RPCServ) SelectUTXO(gctx context.Context, req *protos2.UtxoInput) (*protos2.UtxoOutput, error) {
	// 默认响应
	resp := &protos2.UtxoOutput{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetTotalNeed() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}
	totalNeed, ok := new(big.Int).SetString(req.GetTotalNeed(), 10)
	if !ok {
		rctx.GetLog().Warn("param error,total need set error", "totalNeed", req.GetTotalNeed())
		return resp, engineBase.ErrParameter
	}

	// select utxo
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	out, err := handle.SelectUtxo(req.GetAddress(), totalNeed, req.GetNeedLock(), false,
		req.GetPublickey(), req.GetUserSign())
	if err != nil {
		rctx.GetLog().Warn("select utxo failed", "err", err.Error())
		return resp, err
	}

	utxoList, err := scom.UtxoListToXchain(out.GetUtxoList())
	if err != nil {
		rctx.GetLog().Warn("convert utxo failed", "err", err)
		return resp, engineBase.ErrInternal
	}

	resp.UtxoList = utxoList
	resp.TotalSelected = out.GetTotalSelected()
	return resp, nil
}

// SelectUTXOBySize select utxo inputs depending on size
func (t *RPCServ) SelectUTXOBySize(gctx context.Context, req *protos2.UtxoInput) (*protos2.UtxoOutput, error) {
	// 默认响应
	resp := &protos2.UtxoOutput{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	// select utxo
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	out, err := handle.SelectUTXOBySize(req.GetAddress(), req.GetNeedLock(), false,
		req.GetPublickey(), req.GetUserSign())
	if err != nil {
		rctx.GetLog().Warn("select utxo failed", "err", err.Error())
		return resp, err
	}

	utxoList, err := scom.UtxoListToXchain(out.GetUtxoList())
	if err != nil {
		rctx.GetLog().Warn("convert utxo failed", "err", err)
		return resp, engineBase.ErrInternal
	}

	resp.UtxoList = utxoList
	resp.TotalSelected = out.GetTotalSelected()
	return resp, nil
}

// QueryContractStatData query statistic info about contract
func (t *RPCServ) QueryContractStatData(gctx context.Context,
	req *protos2.ContractStatDataRequest) (*protos2.ContractStatDataResponse, error) {
	// 默认响应
	resp := &protos2.ContractStatDataResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	res, err := handle.QueryContractStatData()
	if err != nil {
		rctx.GetLog().Warn("query contract stat data failed", "err", err.Error())
		return resp, err
	}

	resp.Bcname = req.GetBcname()
	resp.Data = &protos2.ContractStatData{
		AccountCount:  res.GetAccountCount(),
		ContractCount: res.GetContractCount(),
	}

	return resp, nil
}

// QueryUtxoRecord query utxo records
func (t *RPCServ) QueryUtxoRecord(gctx context.Context,
	req *protos2.UtxoRecordDetail) (*protos2.UtxoRecordDetail, error) {

	// 默认响应
	resp := &protos2.UtxoRecordDetail{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	res, err := handle.QueryUtxoRecord(req.GetAccountName(), req.GetDisplayCount())
	if err != nil {
		rctx.GetLog().Warn("query utxo record failed", "err", err.Error())
		return resp, err
	}

	resp.Bcname = req.GetBcname()
	resp.AccountName = req.GetAccountName()
	resp.OpenUtxoRecord = scom.UtxoRecordToXchain(res.GetOpenUtxo())
	resp.LockedUtxoRecord = scom.UtxoRecordToXchain(res.GetLockedUtxo())
	resp.FrozenUtxoRecord = scom.UtxoRecordToXchain(res.GetFrozenUtxo())
	resp.DisplayCount = req.GetDisplayCount()

	return resp, nil
}

// QueryACL query some account info
func (t *RPCServ) QueryACL(gctx context.Context, req *protos2.AclStatus) (*protos2.AclStatus, error) {
	// 默认响应
	resp := &protos2.AclStatus{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}
	if len(req.GetAccountName()) < 1 && (len(req.GetContractName()) < 1 || len(req.GetMethodName()) < 1) {
		rctx.GetLog().Warn("param error,unset name")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}

	var aclRes *protos2.Acl
	if len(req.GetAccountName()) > 0 {
		aclRes, err = handle.QueryAccountACL(req.GetAccountName())
	} else if len(req.GetContractName()) > 0 && len(req.GetMethodName()) > 0 {
		aclRes, err = handle.QueryContractMethodACL(req.GetContractName(), req.GetMethodName())
	}
	if err != nil {
		rctx.GetLog().Warn("query acl failed", "err", err)
		return resp, err
	}

	if aclRes == nil {
		resp.Confirmed = false
		return resp, nil
	}

	xchainAcl := scom.AclToXchain(aclRes)
	if xchainAcl == nil {
		rctx.GetLog().Warn("convert acl failed")
		return resp, engineBase.ErrInternal
	}

	resp.AccountName = req.GetAccountName()
	resp.ContractName = req.GetContractName()
	resp.MethodName = req.GetMethodName()
	resp.Confirmed = true
	resp.Acl = xchainAcl

	return resp, nil
}

// GetAccountContracts get account request
func (t *RPCServ) GetAccountContracts(gctx context.Context, req *protos2.GetAccountContractsRequest) (*protos2.GetAccountContractsResponse, error) {
	// 默认响应
	resp := &protos2.GetAccountContractsResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetAccount() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}

	var res []*protos2.ContractStatus
	res, err = handle.GetAccountContracts(req.GetAccount())
	if err != nil {
		rctx.GetLog().Warn("get account contract failed", "err", err)
		return resp, err
	}
	xchainContractStatus, err := scom.ContractStatusListToXchain(res)
	if xchainContractStatus == nil {
		rctx.GetLog().Warn("convert acl failed")
		return resp, engineBase.ErrInternal
	}

	resp.ContractsStatus = xchainContractStatus

	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("account", req.GetAccount())
	return resp, nil
}

// QueryTx Get transaction details
func (t *RPCServ) QueryTx(gctx context.Context, req *protos2.TxStatus) (*protos2.TxStatus, error) {
	// 默认响应
	resp := &protos2.TxStatus{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || len(req.GetTxid()) == 0 {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err)
		return resp, engineBase.ErrInternal.More("%v", err)
	}

	txInfo, err := handle.QueryTx(req.GetTxid())
	if err != nil {
		rctx.GetLog().Warn("query tx failed", "err", err)
		return resp, err
	}

	tx := scom.TxToXchain(txInfo.Tx)
	if tx == nil {
		rctx.GetLog().Warn("convert tx failed")
		return resp, engineBase.ErrInternal
	}
	resp.Bcname = req.GetBcname()
	resp.Txid = req.GetTxid()
	resp.Tx = tx
	resp.Status = protos2.TransactionStatus(txInfo.Status)
	resp.Distance = txInfo.Distance

	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("account", utils.F(req.GetTxid()))
	return resp, nil
}

// GetBalance get balance for account or addr
func (t *RPCServ) GetBalance(gctx context.Context, req *protos2.AddressStatus) (*protos2.AddressStatus, error) {
	// 默认响应
	resp := &protos2.AddressStatus{
		Bcs: make([]*protos2.TokenDetail, 0),
	}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	for i := 0; i < len(req.Bcs); i++ {
		tmpTokenDetail := &protos2.TokenDetail{}
		handle, err := models.NewChainHandle(req.Bcs[i].Bcname, rctx)
		if err != nil {
			tmpTokenDetail.Error = protos2.XChainErrorEnum_BLOCKCHAIN_NOTEXIST
			tmpTokenDetail.Balance = ""
			resp.Bcs = append(resp.Bcs, tmpTokenDetail)
			continue
		}
		balance, err := handle.GetBalance(req.Address)
		if err != nil {
			tmpTokenDetail.Error = protos2.XChainErrorEnum_UNKNOW_ERROR
			tmpTokenDetail.Balance = ""
		} else {
			tmpTokenDetail.Error = protos2.XChainErrorEnum_SUCCESS
			tmpTokenDetail.Balance = balance
		}
		resp.Bcs = append(resp.Bcs, tmpTokenDetail)
	}
	resp.Address = req.GetAddress()

	rctx.GetLog().SetInfoField("account", req.GetAddress())
	return resp, nil
}

// GetFrozenBalance get balance frozened for account or addr
func (t *RPCServ) GetFrozenBalance(gctx context.Context, req *protos2.AddressStatus) (*protos2.AddressStatus, error) {
	// 默认响应
	resp := &protos2.AddressStatus{
		Bcs: make([]*protos2.TokenDetail, 0),
	}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	for i := 0; i < len(req.Bcs); i++ {
		tmpTokenDetail := &protos2.TokenDetail{
			Bcname: req.Bcs[i].Bcname,
		}
		handle, err := models.NewChainHandle(req.Bcs[i].Bcname, rctx)
		if err != nil {
			tmpTokenDetail.Error = protos2.XChainErrorEnum_BLOCKCHAIN_NOTEXIST
			tmpTokenDetail.Balance = ""
			resp.Bcs = append(resp.Bcs, tmpTokenDetail)
			continue
		}
		balance, err := handle.GetFrozenBalance(req.Address)
		if err != nil {
			tmpTokenDetail.Error = protos2.XChainErrorEnum_UNKNOW_ERROR
			tmpTokenDetail.Balance = ""
		} else {
			tmpTokenDetail.Error = protos2.XChainErrorEnum_SUCCESS
			tmpTokenDetail.Balance = balance
		}
		resp.Bcs = append(resp.Bcs, tmpTokenDetail)
	}
	resp.Address = req.GetAddress()

	rctx.GetLog().SetInfoField("account", req.GetAddress())
	return resp, nil
}

// GetBalanceDetail get balance frozened for account or addr
func (t *RPCServ) GetBalanceDetail(gctx context.Context, req *protos2.AddressBalanceStatus) (*protos2.AddressBalanceStatus, error) {
	// 默认响应
	resp := &protos2.AddressBalanceStatus{
		Tfds: make([]*protos2.TokenFrozenDetails, 0),
	}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	for i := 0; i < len(req.Tfds); i++ {
		tmpFrozenDetails := &protos2.TokenFrozenDetails{
			Bcname: req.Tfds[i].Bcname,
		}
		handle, err := models.NewChainHandle(req.Tfds[i].Bcname, rctx)
		if err != nil {
			tmpFrozenDetails.Error = protos2.XChainErrorEnum_BLOCKCHAIN_NOTEXIST
			tmpFrozenDetails.Tfd = nil
			resp.Tfds = append(resp.Tfds, tmpFrozenDetails)
			continue
		}
		tfd, err := handle.GetBalanceDetail(req.GetAddress())
		if err != nil {
			tmpFrozenDetails.Error = protos2.XChainErrorEnum_UNKNOW_ERROR
			tmpFrozenDetails.Tfd = nil
		} else {
			xchainTfd, err := scom.BalanceDetailsToXchain(tfd)
			if err != nil {
				tmpFrozenDetails.Error = protos2.XChainErrorEnum_UNKNOW_ERROR
				tmpFrozenDetails.Tfd = nil
			}
			tmpFrozenDetails.Error = protos2.XChainErrorEnum_SUCCESS
			tmpFrozenDetails.Tfd = xchainTfd
		}
		resp.Tfds = append(resp.Tfds, tmpFrozenDetails)
	}
	resp.Address = req.GetAddress()

	rctx.GetLog().SetInfoField("account", req.GetAddress())
	return resp, nil
}

// GetBlock get block info according to blockID
func (t *RPCServ) GetBlock(gctx context.Context, req *protos2.BlockID) (*protos2.Block, error) {
	// 默认响应
	resp := &protos2.Block{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || len(req.GetBlockid()) == 0 {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}

	blockInfo, err := handle.QueryBlock(req.GetBlockid(), true)
	if err != nil {
		rctx.GetLog().Warn("query block error", "error", err)
		return resp, err
	}

	block := scom.BlockToXchain(blockInfo.Block)
	if block == nil {
		rctx.GetLog().Warn("convert block failed")
		return resp, engineBase.ErrInternal
	}

	resp.Block = block
	resp.Status = protos2.Block_EBlockStatus(blockInfo.Status)
	resp.Bcname = req.Bcname
	resp.Blockid = req.Blockid

	rctx.GetLog().SetInfoField("blockid", req.GetBlockid())
	rctx.GetLog().SetInfoField("height", blockInfo.GetBlock().GetHeight())
	return resp, nil
}

// GetBlockChainStatus get systemstatus
func (t *RPCServ) GetBlockChainStatus(gctx context.Context, req *protos2.BCStatus) (*protos2.BCStatus, error) {
	// 默认响应
	resp := &protos2.BCStatus{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}

	status, err := handle.QueryChainStatus()
	if err != nil {
		rctx.GetLog().Warn("get chain status error", "error", err)
		return resp, err
	}

	block := scom.BlockToXchain(status.Block)
	if block == nil {
		rctx.GetLog().Warn("convert block failed")
		return resp, err
	}
	ledgerMeta := scom.LedgerMetaToXchain(status.LedgerMeta)
	if ledgerMeta == nil {
		rctx.GetLog().Warn("convert ledger meta failed")
		return resp, err
	}
	utxoMeta := scom.UtxoMetaToXchain(status.UtxoMeta)
	if utxoMeta == nil {
		rctx.GetLog().Warn("convert utxo meta failed")
		return resp, err
	}
	resp.Bcname = req.Bcname
	resp.Meta = ledgerMeta
	resp.Block = block
	resp.UtxoMeta = utxoMeta
	resp.BranchBlockid = status.BranchIds

	rctx.GetLog().SetInfoField("bc_name", req.GetBcname())
	rctx.GetLog().SetInfoField("blockid", utils.F(resp.Block.Blockid))
	return resp, nil
}

// ConfirmBlockChainStatus confirm is_trunk
func (t *RPCServ) ConfirmBlockChainStatus(gctx context.Context, req *protos2.BCStatus) (*protos2.BCTipStatus, error) {
	// 默认响应
	resp := &protos2.BCTipStatus{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}

	isTrunkTip, err := handle.IsTrunkTipBlock(req.GetBlock().GetBlockid())
	if err != nil {
		rctx.GetLog().Warn("query is trunk tip block fail", "err", err.Error())
		return resp, err
	}

	resp.IsTrunkTip = isTrunkTip
	rctx.GetLog().SetInfoField("blockid", utils.F(req.GetBlock().GetBlockid()))
	rctx.GetLog().SetInfoField("is_trunk_tip", isTrunkTip)

	return resp, nil
}

// GetBlockChains get BlockChains
func (t *RPCServ) GetBlockChains(gctx context.Context, req *protos2.CommonIn) (*protos2.BlockChains, error) {
	// 默认响应
	resp := &protos2.BlockChains{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}
	resp.Blockchains = t.engine.GetChains()
	return resp, nil
}

// GetSystemStatus get systemstatus
func (t *RPCServ) GetSystemStatus(gctx context.Context, req *protos2.CommonIn) (*protos2.SystemsStatusReply, error) {
	// 默认响应
	resp := &protos2.SystemsStatusReply{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	systemsStatus := &protos2.SystemsStatus{
		Speeds: &protos2.Speeds{
			SumSpeeds: make(map[string]float64),
			BcSpeeds:  make(map[string]*protos2.BCSpeeds),
		},
	}
	bcs := t.engine.GetChains()
	for _, bcName := range bcs {
		bcStatus := &protos2.BCStatus{Header: req.Header, Bcname: bcName}
		status, err := t.GetBlockChainStatus(gctx, bcStatus)
		if err != nil {
			rctx.GetLog().Warn("get chain status error", "error", err)
		}

		systemsStatus.BcsStatus = append(systemsStatus.BcsStatus, status)
	}

	if req.ViewOption == protos2.ViewOption_NONE || req.ViewOption == protos2.ViewOption_PEERS {
		peerInfo := t.engine.Context().Net.PeerInfo()
		peerUrls := scom.PeerInfoToStrings(peerInfo)
		systemsStatus.PeerUrls = peerUrls
	}

	resp.SystemsStatus = systemsStatus
	return resp, nil
}

// GetNetURL get net url in p2p_base
func (t *RPCServ) GetNetURL(gctx context.Context, req *protos2.CommonIn) (*protos2.RawUrl, error) {
	// 默认响应
	resp := &protos2.RawUrl{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}
	peerInfo := t.engine.Context().Net.PeerInfo()
	resp.RawUrl = peerInfo.Address

	rctx.GetLog().SetInfoField("raw_url", resp.RawUrl)
	return resp, nil
}

// GetBlockByHeight  get trunk block by height
func (t *RPCServ) GetBlockByHeight(gctx context.Context, req *protos2.BlockHeight) (*protos2.Block, error) {
	// 默认响应
	resp := &protos2.Block{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}
	blockInfo, err := handle.QueryBlockByHeight(req.GetHeight(), true)
	if err != nil {
		rctx.GetLog().Warn("query block error", "bc", req.GetBcname(), "height", req.GetHeight())
		return resp, err
	}

	block := scom.BlockToXchain(blockInfo.Block)
	if block == nil {
		rctx.GetLog().Warn("convert block failed")
		return resp, engineBase.ErrInternal
	}
	resp.Block = block
	resp.Status = protos2.Block_EBlockStatus(blockInfo.Status)
	resp.Bcname = req.GetBcname()
	resp.Blockid = blockInfo.Block.Blockid

	rctx.GetLog().SetInfoField("height", req.GetHeight())
	rctx.GetLog().SetInfoField("blockid", utils.F(blockInfo.Block.Blockid))
	return resp, nil
}

// GetAccountByAK get account list with contain ak
func (t *RPCServ) GetAccountByAK(gctx context.Context, req *protos2.AK2AccountRequest) (*protos2.AK2AccountResponse, error) {
	// 默认响应
	resp := &protos2.AK2AccountResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}

	accounts, err := handle.GetAccountByAK(req.GetAddress())
	if err != nil || accounts == nil {
		rctx.GetLog().Warn("QueryAccountContainAK error", "error", err)
		return resp, err
	}

	resp.Account = accounts
	resp.Bcname = req.GetBcname()

	rctx.GetLog().SetInfoField("address", req.GetAddress())
	return resp, err
}

// GetAddressContracts get contracts of accounts contain a specific address
func (t *RPCServ) GetAddressContracts(gctx context.Context, req *protos2.AddressContractsRequest) (*protos2.AddressContractsResponse, error) {
	// 默认响应
	resp := &protos2.AddressContractsResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, err
	}

	accounts, err := handle.GetAccountByAK(req.GetAddress())
	if err != nil || accounts == nil {
		rctx.GetLog().Warn("GetAccountByAK error", "error", err)
		return resp, err
	}

	// get contracts for each account
	resp.Contracts = make(map[string]*protos2.ContractList)
	for _, account := range accounts {
		contracts, err := handle.GetAccountContracts(account)
		if err != nil {
			rctx.GetLog().Warn("GetAddressContracts partial account error", "logid", req.Header.Logid, "error", err)
			continue
		}

		if len(contracts) > 0 {
			xchainContracts, err := scom.ContractStatusListToXchain(contracts)
			if err != nil || xchainContracts == nil {
				rctx.GetLog().Warn("convert contracts failed")
				continue
			}
			resp.Contracts[account] = &protos2.ContractList{
				ContractStatus: xchainContracts,
			}
		}
	}

	rctx.GetLog().SetInfoField("address", req.GetAddress())
	return resp, nil
}

func (t *RPCServ) GetConsensusStatus(gctx context.Context, req *protos2.ConsensusStatRequest) (*protos2.ConsensusStatus, error) {
	// 默认响应
	resp := &protos2.ConsensusStatus{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}
	handle, err := models.NewChainHandle(req.GetBcname(), rctx)
	if err != nil {
		rctx.GetLog().Warn("new chain handle failed", "err", err.Error())
		return resp, engineBase.ErrForbidden
	}

	status, err := handle.QueryConsensusStatus()
	if err != nil {
		rctx.GetLog().Warn("get chain status error", "err", err)
		return resp, engineBase.ErrForbidden
	}
	resp.ConsensusName = status.ConsensusName
	resp.Version = status.Version
	resp.StartHeight = status.StartHeight
	resp.ValidatorsInfo = status.ValidatorsInfo
	return resp, nil
}

// DposCandidates get all candidates of the xpos consensus
func (t *RPCServ) DposCandidates(gctx context.Context, req *protos2.DposCandidatesRequest) (*protos2.DposCandidatesResponse, error) {
	// 默认响应
	resp := &protos2.DposCandidatesResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	return resp, engineBase.ErrForbidden
}

// DposNominateRecords get all records nominated by an user
func (t *RPCServ) DposNominateRecords(gctx context.Context, req *protos2.DposNominateRecordsRequest) (*protos2.DposNominateRecordsResponse, error) {
	// 默认响应
	resp := &protos2.DposNominateRecordsResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	return resp, engineBase.ErrForbidden
}

// DposNomineeRecords get nominated record of a candidate
func (t *RPCServ) DposNomineeRecords(gctx context.Context, req *protos2.DposNomineeRecordsRequest) (*protos2.DposNomineeRecordsResponse, error) {
	// 默认响应
	resp := &protos2.DposNomineeRecordsResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	return resp, engineBase.ErrForbidden
}

// DposVoteRecords get all vote records voted by an user
func (t *RPCServ) DposVoteRecords(gctx context.Context, req *protos2.DposVoteRecordsRequest) (*protos2.DposVoteRecordsResponse, error) {
	// 默认响应
	resp := &protos2.DposVoteRecordsResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	return resp, engineBase.ErrForbidden
}

// DposVotedRecords get all vote records of a candidate
func (t *RPCServ) DposVotedRecords(gctx context.Context, req *protos2.DposVotedRecordsRequest) (*protos2.DposVotedRecordsResponse, error) {
	// 默认响应
	resp := &protos2.DposVotedRecordsResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" || req.GetAddress() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	return resp, engineBase.ErrForbidden
}

// DposCheckResults get check results of a specific term
func (t *RPCServ) DposCheckResults(gctx context.Context, req *protos2.DposCheckResultsRequest) (*protos2.DposCheckResultsResponse, error) {
	// 默认响应
	resp := &protos2.DposCheckResultsResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	return resp, engineBase.ErrForbidden
}

// DposStatus get dpos status
func (t *RPCServ) DposStatus(gctx context.Context, req *protos2.DposStatusRequest) (*protos2.DposStatusResponse, error) {
	// 默认响应
	resp := &protos2.DposStatusResponse{}
	// 获取请求上下文，对内传递rctx
	rctx := sctx.ValueReqCtx(gctx)

	if req == nil || req.GetBcname() == "" {
		rctx.GetLog().Warn("param error,some param unset")
		return resp, engineBase.ErrParameter
	}

	return resp, engineBase.ErrForbidden
}
