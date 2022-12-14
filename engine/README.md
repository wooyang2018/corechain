# 区块链引擎

区块链引擎：定义一种区块链核心流程实现。

内核框架采用多引擎架构，每个引擎订制一套区块链内核实现，所有引擎注册到引擎工厂，外部通过工厂实例化引擎。
每个引擎提供执行引擎和读组件两部分能力。各引擎间交易、区块结构无关，共用内核核心组件。

## 引擎介绍

xuperos: 面向公链场景区块链网络内核实现。

xchain: 面向联盟联盟场景区块链网络内核实现。

## 使用示例

```

// 加载内核运行环境配置
envCfgPath := "/home/rd/xx/conf/env.yaml"
envCfg, _ := engines.LoadEnvConf(envCfgPath)

// 创建内核引擎实例
engine, _ := engines.CreateBCEngine("xchain", envCfg)

engine.Init()
engine.Start()
engine.Stop()

xEngine, _ := xuperos.EngineConvert(engine)
xChain := xEngine.Get("xuper")
xChain.PreExec()
xChain.ProcessTx()
```



# xuperos

面向公链应用场景设计的区块链执行引擎。

## 组件说明

reader: 只读组件。采用读写分离设计，降低代码复杂度。

ledger: 统一收敛账本变更操作（交易、同步块、创世块修改等）。

contract: 系统合约，考虑到系统合约和链强相关，放到引擎中实现，注册到合约组件。

## 应用案例

超级链开放网络。

## 对外接口

```

type BCEngine interface {
	// 初始化引擎
	Init(*xconf.EnvConf) error
	// 启动引擎(阻塞)
	Run()
	// 退出引擎，需要幂等
	Exit()
}

type Engine interface {
	BCEngine
	Context() *EngineCtx
	Get(string) Chain
	GetChains() []string
	SetRelyAgent(EngineRelyAgent) error
}

type Chain interface {
	Context() *ChainCtx
	Start()
	Stop()
	// 合约预执行
	PreExec(xctx.Context, []*protos.InvokeRequest) (*protos.InvokeResponse, error)
	// 提交交易
	SubmitTx(xctx.Context, *lpb.Transaction) error
	// 设置依赖实例化代理
	SetRelyAgent(ChainRelyAgent) error
}

type ChainReader interface {
	// 获取链状态 (GetBlockChainStatus)
	GetChainStatus() (*ChainStatus, error)
	// 检查是否是主干Tip Block (ConfirmBlockChainStatus)
	IsTrunkTipBlock(blkId []byte) (bool, error)
	// 获取系统状态
	GetSystemStatus() (*ChainStatus, error)
	// 获取节点NetUR
	GetNetURL() (string, error)
}

type ConsensusReader interface {
	// 获取共识状态
	GetConsStatus() (consBase.ConsensusStatus, error)
}

type ContractReader interface {
	// 查询该链合约统计数据
	QueryContractStatData() (*protos.ContractStatData, error)
	// 查询账户下合约状态
	GetAccountContracts(account string) ([]*protos.ContractStatus, error)
	// 查询地址下合约状态
	GetAddressContracts(address string, needContent bool) (map[string][]*protos.ContractStatus, error)
	// 查询地址下账户
	GetAccountByAK(address string) ([]string, error)
	// 查询合约账户ACL
	QueryAccountACL(account string) (*protos.Acl, bool, error)
	// 查询合约方法ACL
	QueryContractMethodACL(contract, method string) (*protos.Acl, bool, error)
}

type LedgerReader interface {
	// 查询交易信息（QueryTx）
	QueryTx(txId []byte) (*lpb.TxInfo, error)
	// 查询区块ID信息（GetBlock）
	QueryBlock(blkId []byte, needContent bool) (*lpb.BlockInfo, error)
	// 通过区块高度查询区块信息（GetBlockByHeight）
	QueryBlockByHeight(height int64, needContent bool) (*lpb.BlockInfo, error)
}

type UtxoReader interface {
	// 获取账户余额
	GetBalance(account string) (string, error)
	// 获取账户冻结余额
	GetFrozenBalance(account string) (string, error)
	// 获取账户余额详情
	GetBalanceDetail(account string) ([]*lpb.BalanceDetailInfo, error)
	// 拉取固定数目的utxo
	QueryUtxoRecord(account string, count int64) (*lpb.UtxoRecordDetail, error)
	// 按最大交易大小选择utxo
	SelectUTXOBySize(account string, isLock, isExclude bool) (*lpb.UtxoOutput, error)
	// 选择合适金额的utxo
	SelectUTXO(account string, need *big.Int, isLock, isExclude bool) (*lpb.UtxoOutput, error)
}

```


