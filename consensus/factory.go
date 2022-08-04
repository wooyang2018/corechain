package consensus

import "github.com/wooyang2018/corechain/consensus/base"

//consensusMap 名称和新建具体共识实例的函数的映射
var consensusMap = make(map[string]NewStepConsensus)

type NewStepConsensus func(cctx base.ConsensusCtx, ccfg base.ConsensusConfig) base.CommonConsensus

// Register 注册共识协议
func Register(name string, f NewStepConsensus) error {
	if f == nil {
		panic("Pluggable CommonConsensus::Register::new function is nil")
	}
	if _, dup := consensusMap[name]; dup {
		panic("Pluggable CommonConsensus::Register::called twice for func " + name)
	}
	consensusMap[name] = f
	return nil
}

// NewPluginConsensus 新建可插拔共识实例
func NewPluginConsensus(cctx base.ConsensusCtx, ccfg base.ConsensusConfig) (base.CommonConsensus, error) {
	if ccfg.ConsensusName == "" {
		return nil, EmptyConsensusName
	}
	if ccfg.StartHeight < 0 {
		return nil, BeginBlockIdErr
	}
	if f, ok := consensusMap[ccfg.ConsensusName]; ok {
		return f(cctx, ccfg), nil
	}
	return nil, ConsensusNotRegister
}
