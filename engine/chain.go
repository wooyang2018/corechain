package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/wooyang2018/corechain/common/address"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/metrics"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/common/utils"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/engine/agent"
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/engine/miner"
	"github.com/wooyang2018/corechain/engine/parachain"
	ltx "github.com/wooyang2018/corechain/ledger/tx"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state"
	"github.com/wooyang2018/corechain/state/model"
)

const (
	// 提交交易cache有效期(s)
	TxIdCacheExpired = 120 * time.Second
	// 提交交易cache GC 周期（s）
	TxIdCacheGCInterval = 180 * time.Second
)

// 定义一条链的具体行为，对外暴露接口错误统一使用标准错误
type Chain struct {
	// 链上下文
	ctx *base.ChainCtx
	// log
	log logger.Logger
	// 矿工
	miner *miner.Miner
	// 依赖代理组件
	relyAgent base.ChainRelyAgent
	// 提交交易cache
	txIdCache *cache.Cache
}

// 从本地存储加载链
func LoadChain(engCtx *base.EngineCtx, bcName string) (*Chain, error) {
	if engCtx == nil || bcName == "" {
		return nil, base.ErrParameter
	}

	// 实例化链日志句柄
	log, err := logger.NewLogger("", "engine")
	if err != nil {
		return nil, base.ErrNewLogFailed
	}

	// 实例化链实例
	ctx := &base.ChainCtx{}
	ctx.EngCtx = engCtx
	ctx.BcName = bcName
	ctx.XLog = log
	ctx.Timer = timer.NewXTimer()
	chainObj := &Chain{}
	chainObj.ctx = ctx
	chainObj.log = ctx.XLog
	chainObj.relyAgent = agent.NewChainRelyAgent(ctx)

	// 初始化链运行环境上下文
	err = chainObj.initChainCtx()
	if err != nil {
		log.Error("init chain ctx failed", "bcName", bcName, "err", err)
		return nil, base.ErrNewChainCtxFailed.More("err:%v", err)
	}

	// 创建矿工
	chainObj.miner = miner.NewMiner(ctx)
	chainObj.txIdCache = cache.New(TxIdCacheExpired, TxIdCacheGCInterval)

	return chainObj, nil
}

// 供单测时设置rely agent为mock agent，非并发安全 未使用到的函数
func (t *Chain) SetRelyAgent(agent base.ChainRelyAgent) error {
	if agent == nil {
		return base.ErrParameter
	}

	t.relyAgent = agent
	return nil
}

// 阻塞
func (t *Chain) Start() {
	// 启动矿工
	t.miner.Start()
}

func (t *Chain) Stop() {
	// 停止矿工等其余组件
	t.miner.Stop()
	t.ctx.Ledger.Close()
	t.ctx.State.Close()
	t.ctx = nil
	t.miner = nil
	t.txIdCache = nil
}

func (t *Chain) Context() *base.ChainCtx {
	return t.ctx
}

// 交易预执行
func (t *Chain) PreExec(ctx xctx.Context, reqs []*protos.InvokeRequest, initiator string, authRequires []string) (*protos.InvokeResponse, error) {
	if ctx == nil || ctx.GetLog() == nil {
		return nil, base.ErrParameter
	}

	reservedRequests, err := t.ctx.State.GetReservedContractRequests(reqs, true)
	if err != nil {
		t.log.Error("PreExec get reserved contract request error", "error", err)
		return nil, base.ErrParameter.More("%v", err)
	}

	transContractName, transAmount, err := ltx.ParseContractTransferRequest(reqs)
	if err != nil {
		return nil, base.ErrParameter.More("%v", err)
	}

	reqs = append(reservedRequests, reqs...)
	if len(reqs) <= 0 {
		return &protos.InvokeResponse{}, nil
	}

	stateConfig := &contractBase.SandboxConfig{
		XMReader:   t.ctx.State.CreateXMReader(),
		UTXOReader: t.ctx.State.CreateUtxoReader(),
	}
	sandbox, err := t.ctx.Contract.NewStateSandbox(stateConfig)
	if err != nil {
		t.log.Error("PreExec new state sandbox error", "error", err)
		return nil, base.ErrContractNewSandboxFailed
	}

	contextConfig := &contractBase.ContextConfig{
		State:          sandbox,
		Initiator:      initiator,
		AuthRequire:    authRequires,
		ResourceLimits: contractBase.MaxLimits,
	}

	gasPrice := t.ctx.State.GetMeta().GetGasPrice()
	gasUsed := int64(0)
	responseBodes := make([][]byte, 0, len(reqs))
	requests := make([]*protos.InvokeRequest, 0, len(reqs))
	responses := make([]*protos.ContractResponse, 0, len(reqs))
	for i, req := range reqs {
		if req == nil {
			continue
		}

		if req.ModuleName == "" && req.ContractName == "" && req.MethodName == "" {
			ctx.GetLog().Warn("PreExec req empty", "req", req)
			continue
		}

		beginTime := time.Now()
		contextConfig.Module = req.ModuleName
		contextConfig.ContractName = req.ContractName
		if transContractName == req.ContractName {
			contextConfig.TransferAmount = transAmount.String()
		} else {
			contextConfig.TransferAmount = ""
		}

		context, err := t.ctx.Contract.NewContext(contextConfig)
		if err != nil {
			ctx.GetLog().Error("PreExec NewContext error", "error", err, "contractName", req.ContractName)
			if i < len(reservedRequests) && strings.HasSuffix(err.Error(), "not found") {
				requests = append(requests, req)
				continue
			}
			return nil, base.ErrContractNewCtxFailed.More("%v", err)
		}

		resp, err := context.Invoke(req.MethodName, req.Args)
		if err != nil {
			context.Release()
			ctx.GetLog().Error("PreExec Invoke error", "error", err, "contractName", req.ContractName)
			metrics.ContractInvokeCounter.WithLabelValues(t.ctx.BcName, req.ModuleName, req.ContractName, req.MethodName, "InvokeError").Inc()
			return nil, base.ErrContractInvokeFailed.More("%v", err)
		}

		if resp.Status >= 400 && i < len(reservedRequests) {
			context.Release()
			ctx.GetLog().Error("PreExec Invoke error", "status", resp.Status, "contractName", req.ContractName)
			metrics.ContractInvokeCounter.WithLabelValues(t.ctx.BcName, req.ModuleName, req.ContractName, req.MethodName, "InvokeError").Inc()
			return nil, base.ErrContractInvokeFailed.More("%v", resp.Message)
		}

		metrics.ContractInvokeCounter.WithLabelValues(t.ctx.BcName, req.ModuleName, req.ContractName, req.MethodName, "OK").Inc()
		resourceUsed := context.ResourceUsed()
		if i >= len(reservedRequests) {
			gasUsed += resourceUsed.TotalGas(gasPrice)
		}

		// request
		request := *req
		request.ResourceLimits = contractBase.ToPbLimits(resourceUsed)
		requests = append(requests, &request)

		// response
		response := &protos.ContractResponse{
			Status:  int32(resp.Status),
			Message: resp.Message,
			Body:    resp.Body,
		}
		responses = append(responses, response)
		responseBodes = append(responseBodes, resp.Body)

		context.Release()
		metrics.ContractInvokeHistogram.WithLabelValues(t.ctx.BcName, req.ModuleName, req.ContractName, req.MethodName).Observe(time.Since(beginTime).Seconds())
	}

	err = sandbox.Flush()
	if err != nil {
		return nil, err
	}
	rwSet := sandbox.RWSet()
	utxoRWSet := sandbox.UTXORWSet()

	invokeResponse := &protos.InvokeResponse{
		GasUsed:     gasUsed,
		Response:    responseBodes,
		Inputs:      model.GetTxInputs(rwSet.RSet),
		Outputs:     model.GetTxOutputs(rwSet.WSet),
		Requests:    requests,
		Responses:   responses,
		UtxoInputs:  utxoRWSet.Rset,
		UtxoOutputs: utxoRWSet.WSet,
	}

	return invokeResponse, nil
}

// 提交交易到交易池(xuperos引擎同时更新到状态机和交易池)
func (t *Chain) SubmitTx(ctx xctx.Context, tx *protos.Transaction) error {
	if tx == nil || ctx == nil || ctx.GetLog() == nil || len(tx.GetTxid()) <= 0 {
		return base.ErrParameter
	}
	log := ctx.GetLog()

	// 无币化
	if len(tx.TxInputs) == 0 && !t.ctx.Ledger.GetNoFee() {
		ctx.GetLog().Warn("PostTx TxInputs can not be null while need utxo")
		return base.ErrTxNotEnough
	}

	// 防止重复提交交易
	if _, exist := t.txIdCache.Get(string(tx.GetTxid())); exist {
		return base.ErrTxAlreadyExist
	}
	t.txIdCache.Set(string(tx.GetTxid()), true, TxIdCacheExpired)

	code := "OK"
	defer func() {
		metrics.CallMethodCounter.WithLabelValues(t.ctx.BcName, "SubmitTx", code).Inc()
	}()

	// 判断此交易是否已经存在（账本和未确认交易表中）。
	dbtx, _, _ := t.ctx.State.QueryTx(tx.GetTxid())
	if dbtx != nil { // 从数据库查询到了交易，返回错误。
		log.Error("tx already exist", "txid", utils.F(tx.GetTxid()))
		return base.ErrTxAlreadyExist
	}

	// 验证交易
	_, err := t.ctx.State.VerifyTx(tx)
	if err != nil {
		log.Error("verify tx error", "txid", utils.F(tx.GetTxid()), "err", err)
		code = "VerifyTxFailed"
		return base.ErrTxVerifyFailed.More("err:%v", err)
	}

	// 提交交易
	err = t.ctx.State.DoTx(tx)
	if err != nil {
		log.Error("submit tx error", "txid", utils.F(tx.GetTxid()), "err", err)
		if err == state.ErrAlreadyInUnconfirmed {
			t.txIdCache.Delete(string(tx.GetTxid()))
		}
		code = "SubmitTxFailed"
		return base.ErrSubmitTxFailed.More("err:%v", err)
	}

	return nil
}

// 处理P2P网络同步到的区块
func (t *Chain) ProcBlock(ctx xctx.Context, block *protos.InternalBlock) error {
	if block == nil || ctx == nil || ctx.GetLog() == nil || block.GetBlockid() == nil {
		return base.ErrParameter
	}

	log := ctx.GetLog()
	err := t.miner.ProcBlock(ctx, block)
	if err != nil {
		if base.CastError(err).Equal(base.ErrForbidden) {
			log.Debug("forbidden process block", "blockid", utils.F(block.GetBlockid()), "err", err)
			return base.ErrForbidden
		}

		if base.CastError(err).Equal(base.ErrParameter) {
			log.Debug("param error")
			return base.ErrParameter
		}

		ctx.GetLog().Warn("process block failed", "blockid", utils.F(block.GetBlockid()), "err", err)
		return base.ErrProcBlockFailed.More("err:%v", err)
	}

	log.Info("process block succ", "height", block.GetHeight(), "blockid", utils.F(block.GetBlockid()))
	return nil
}

// 初始化链运行依赖上下文
func (t *Chain) initChainCtx() error {
	// 1.实例化账本
	leg, err := t.relyAgent.CreateLedger()
	if err != nil {
		t.log.Error("open ledger failed", "bcName", t.ctx.BcName, "err", err)
		return err
	}
	t.ctx.Ledger = leg
	t.log.Debug("open ledger succ", "bcName", t.ctx.BcName)

	// 2.实例化加密组件
	// 从账本查询加密算法类型
	cryptoType, err := agent.NewLedgerAgent(t.ctx).GetCryptoType()
	if err != nil {
		t.log.Error("query crypto type failed", "bcName", t.ctx.BcName, "err", err)
		return fmt.Errorf("query crypto type failed")
	}
	crypt, err := t.relyAgent.CreateCrypto(cryptoType)
	if err != nil {
		t.log.Error("create crypto client failed", "error", err)
		return fmt.Errorf("create crypto client failed")
	}
	t.ctx.Crypto = crypt
	t.log.Debug("create crypto client succ", "bcName", t.ctx.BcName, "cryptoType", cryptoType)

	// 3.实例化状态机
	stat, err := t.relyAgent.CreateState(leg, crypt)
	if err != nil {
		t.log.Error("open state failed", "bcName", t.ctx.BcName, "err", err)
		return fmt.Errorf("open state failed")
	}
	t.ctx.State = stat
	t.log.Debug("open state succ", "bcName", t.ctx.BcName)

	// 4.加载节点账户信息
	keyPath := t.ctx.EngCtx.EnvCfg.GenDataAbsPath(t.ctx.EngCtx.EnvCfg.KeyDir)
	addr, err := address.LoadAddrInfo(keyPath, t.ctx.Crypto)
	if err != nil {
		t.log.Error("load node addr info error", "bcName", t.ctx.BcName, "keyPath", keyPath, "err", err)
		return fmt.Errorf("load node addr info error")
	}
	t.ctx.Address = addr
	t.log.Debug("load node addr info succ", "bcName", t.ctx.BcName, "address", addr.Address)

	// 5.合约
	contractObj, err := t.relyAgent.CreateContract(stat.CreateXMReader())
	if err != nil {
		t.log.Error("create contract manager error", "bcName", t.ctx.BcName, "err", err)
		return fmt.Errorf("create contract manager error")
	}
	t.ctx.Contract = contractObj
	// 设置合约manager到状态机
	t.ctx.State.SetContractMG(t.ctx.Contract)
	t.log.Debug("create contract manager succ", "bcName", t.ctx.BcName)

	// 6.Acl
	aclObj, err := t.relyAgent.CreateAcl()
	if err != nil {
		t.log.Error("create acl error", "bcName", t.ctx.BcName, "err", err)
		return fmt.Errorf("create acl error")
	}
	t.ctx.Acl = aclObj
	// 设置acl manager到状态机
	t.ctx.State.SetAclMG(t.ctx.Acl)
	t.log.Debug("create acl succ", "bcName", t.ctx.BcName)

	// 7.共识
	cons, err := t.relyAgent.CreateConsensus()
	if err != nil {
		t.log.Error("create consensus error", "bcName", t.ctx.BcName, "err", err)
		return fmt.Errorf("create consensus error")
	}
	t.ctx.Consensus = cons
	t.log.Debug("create consensus succ", "bcName", t.ctx.BcName)

	// 8.提案
	governTokenObj, err := t.relyAgent.CreateGovernToken()
	if err != nil {
		t.log.Error("create govern token error", "bcName", t.ctx.BcName, "err", err)
		return fmt.Errorf("create govern token error")
	}
	t.ctx.GovernToken = governTokenObj
	// 设置govern token manager到状态机
	t.ctx.State.SetGovernTokenMG(t.ctx.GovernToken)
	t.log.Debug("create govern token succ", "bcName", t.ctx.BcName)

	// 9.提案
	proposalObj, err := t.relyAgent.CreateProposal()
	if err != nil {
		t.log.Error("create proposal error", "bcName", t.ctx.BcName, "err", err)
		return fmt.Errorf("create proposal error")
	}
	t.ctx.Proposal = proposalObj
	// 设置proposal manager到状态机
	t.ctx.State.SetProposalMG(t.ctx.Proposal)
	t.log.Debug("create proposal succ", "bcName", t.ctx.BcName)

	// 10.定时器任务
	timerObj, err := t.relyAgent.CreateTimerTask()
	if err != nil {
		t.log.Error("create timer_task error", "bcName", t.ctx.BcName, "err", err)
		return fmt.Errorf("create timer_task error")
	}
	t.ctx.TimerTask = timerObj
	// 设置timer manager到状态机
	t.ctx.State.SetTimerTaskMG(t.ctx.TimerTask)
	t.log.Debug("create timer_task succ", "bcName", t.ctx.BcName)
	t.log.Debug("create chain succ", "bcName", t.ctx.BcName)
	return nil
}

// 创建平行链实例
func (t *Chain) CreateParaChain() error {
	paraChainCtx, err := parachain.NewParaChainCtx(t.ctx.BcName, t.ctx)
	if err != nil {
		return fmt.Errorf("create parachain ctx failed.err:%v", err)
	}
	_, err = parachain.NewParaChainManager(paraChainCtx)
	if err != nil {
		return fmt.Errorf("create parachain instance failed.err:%v", err)
	}
	return nil
}
