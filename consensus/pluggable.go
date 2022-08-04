package consensus

import (
	"encoding/json"
	"errors"
	"strconv"
	"sync"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/consensus/base"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/ledger"
)

const (
	// contractUpdateMethod 为更新共识注册，用于在提案-投票成功后，触发共识由原A转换成B
	contractUpdateMethod = "updateConsensus"
	// 可插拔共识使用的三代kernel合约存储bucket名
	// <"PluggableConfig", configJson> 其中configJson为一个map[int]consensusJson格式，key为自增index，value为对应共识config
	// <index, consensusJson<STRING>> 每个index对应的共识属性，eg. <"1", "{"name":"pow", "config":"{}", "beginHeight":"100"}">
	contractBucket = "$consensus"
	consensusKey   = "PluggableConfig"
)

var (
	EmptyConsensusListErr = errors.New("CommonConsensus list of PluggableConsensusImpl is empty.")
	EmptyConsensusName    = errors.New("CommonConsensus name can not be empty")
	EmptyConfig           = errors.New("Config name can not be empty")
	UpdateTriggerError    = errors.New("Update trigger height invalid")
	BeginBlockIdErr       = errors.New("CommonConsensus begin blockid err")
	BuildConsensusError   = errors.New("Build consensus Error")
	ConsensusNotRegister  = errors.New("CommonConsensus hasn't been register. Please use consensus.Register({NAME},{FUNCTION_POINTER}) to register in consensusMap")
	ContractMngErr        = errors.New("Contract manager is empty.")

	ErrInvalidConfig  = errors.New("config should be an empty JSON when rolling back an old one, or try an upper version")
	ErrInvalidVersion = errors.New("version should be an upper one when upgrading a new one")
)

// stepConsensus 封装了可插拔共识需要的共识数组
type stepConsensus struct {
	commonConsensuses []base.CommonConsensus
	// 共识升级切换开关
	switchConsensus bool
	mutex           sync.Mutex
}

// 获取共识切换开关
func (sc *stepConsensus) getSwitch() bool {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return sc.switchConsensus
}

// 设置共识切换开关
func (sc *stepConsensus) setSwitch(s bool) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.switchConsensus = s
}

// 向可插拔共识数组put
func (sc *stepConsensus) put(con base.CommonConsensus) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.commonConsensuses = append(sc.commonConsensuses, con)
	return nil
}

// 获取最新的共识实例
func (sc *stepConsensus) tail() base.CommonConsensus {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if len(sc.commonConsensuses) == 0 {
		return nil
	}
	return sc.commonConsensuses[len(sc.commonConsensuses)-1]
}

// 获取倒数第二个共识实例
func (sc *stepConsensus) preTail() base.CommonConsensus {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	if len(sc.commonConsensuses) < 2 {
		return nil
	}
	return sc.commonConsensuses[len(sc.commonConsensuses)-2]
}

// 获取共识实例长度
func (sc *stepConsensus) len() int {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	return len(sc.commonConsensuses)
}

// PluggableConsensusImpl 利用stepConsensus实现可插拔共识
type PluggableConsensusImpl struct {
	ctx           base.ConsensusCtx
	stepConsensus *stepConsensus
}

// NewPluggableConsensus 初次创建PluggableConsensus实例，初始化可插拔共识列表
func NewPluggableConsensus(cctx base.ConsensusCtx) (base.PluggableConsensus, error) {
	if cctx.BcName == "" {
		cctx.XLog.Error("Pluggable CommonConsensus::NewPluggableConsensus::bcName is empty.")
	}
	pc := &PluggableConsensusImpl{
		ctx: cctx,
		stepConsensus: &stepConsensus{
			commonConsensuses: []base.CommonConsensus{},
		},
	}
	if cctx.Contract.GetKernRegistry() == nil {
		return nil, ContractMngErr
	}
	// 向合约注册升级方法
	cctx.Contract.GetKernRegistry().RegisterKernMethod(contractBucket, contractUpdateMethod, pc.updateConsensus)
	xMReader, err := cctx.Ledger.GetTipXMSnapshotReader()
	if err != nil {
		return nil, err
	}
	res, _ := xMReader.Get(contractBucket, []byte(consensusKey))
	// 若合约存储不存在，则证明为第一次创建实例，则直接从账本里拿到创始块配置并启动共识
	if res == nil {
		consensusBuf, err := cctx.Ledger.GetConsensusConf()
		if err != nil {
			return nil, err
		}
		// 解析提取字段生成ConsensusConfig
		cfg := base.ConsensusConfig{}
		err = json.Unmarshal(consensusBuf, &cfg)
		if err != nil {
			cctx.XLog.Error("Pluggable CommonConsensus::NewPluggableConsensus::parse consensus configuration error!", "conf", string(consensusBuf), "error", err.Error())
			return nil, err
		}
		cfg.StartHeight = 1
		cfg.Index = 0
		genesisConsensus, err := pc.makeConsensusItem(cctx, cfg)
		if err != nil {
			cctx.XLog.Error("Pluggable CommonConsensus::NewPluggableConsensus::make first consensus item error!", "error", err.Error())
			return nil, err
		}
		pc.stepConsensus.put(genesisConsensus)
		// 启动实例
		genesisConsensus.Start()
		cctx.XLog.Debug("Pluggable CommonConsensus::NewPluggableConsensus::create a instance for the first time.")
		return pc, nil
	}
	// 原合约存储存在，即该链重启，重新恢复可插拔共识
	c := map[int]base.ConsensusConfig{}
	err = json.Unmarshal(res, &c)
	if err != nil {
		// 历史consensus存储有误，装载无效，此时直接panic
		cctx.XLog.Error("Pluggable CommonConsensus::history consensus storage invalid, pls check function.")
		return nil, err
	}
	for i := 0; i < len(c); i++ {
		config := c[i]
		oldConsensus, err := pc.makeConsensusItem(cctx, config)
		if err != nil {
			cctx.XLog.Warn("Pluggable CommonConsensus::NewPluggableConsensus::make old consensus item error!", "error", err.Error())
		}
		pc.stepConsensus.put(oldConsensus)
		if i == len(c)-1 {
			oldConsensus.Start()
		}
		cctx.XLog.Debug("Pluggable CommonConsensus::NewPluggableConsensus::create a instance with history reader.", "StepConsensus", pc.stepConsensus)
	}

	return pc, nil
}

// makeConsensusItem 创建单个特定共识，返回特定共识接口
func (pc *PluggableConsensusImpl) makeConsensusItem(cctx base.ConsensusCtx, cCfg base.ConsensusConfig) (base.CommonConsensus, error) {
	if cCfg.ConsensusName == "" {
		cctx.XLog.Error("Pluggable CommonConsensus::makeConsensusItem::consensus name is empty")
		return nil, EmptyConsensusName
	}
	specificCon, err := NewPluginConsensus(pc.ctx, cCfg)
	if err != nil {
		cctx.XLog.Error("Pluggable CommonConsensus::NewPluginConsensus error", "error", err)
		return nil, err
	}
	if specificCon == nil {
		cctx.XLog.Error("Pluggable CommonConsensus::NewPluginConsensus::empty error", "error", BuildConsensusError)
		return nil, BuildConsensusError
	}
	cctx.XLog.Debug("Pluggable CommonConsensus::makeConsensusItem::create a consensus item.", "type", cCfg.ConsensusName)
	return specificCon, nil
}

func (pc *PluggableConsensusImpl) proposalArgsUnmarshal(ctxArgs map[string][]byte) (*base.ConsensusConfig, error) {
	if _, ok := ctxArgs["height"]; !ok {
		return nil, UpdateTriggerError
	}
	updateHeight, err := strconv.ParseInt(string(ctxArgs["height"]), 10, 64)
	if err != nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::updateConsensus::height value invalid.", "err", err)
		return nil, err
	}
	args := make(map[string]interface{})
	err = json.Unmarshal(ctxArgs["args"], &args)
	if err != nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::updateConsensus::unmarshal err.", "err", err)
		return nil, err
	}
	if _, ok := args["name"]; !ok {
		return nil, EmptyConsensusName
	}
	if _, ok := args["config"]; !ok {
		return nil, ConsensusNotRegister
	}

	consensusName, ok := args["name"].(string)
	if !ok {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::updateConsensus::name should be string.")
		return nil, EmptyConsensusName
	}
	if _, dup := consensusMap[consensusName]; !dup {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::updateConsensus::consensus's type invalid when update", "name", consensusName)
		return nil, ConsensusNotRegister
	}
	// 解析arg生成用户tx中的共识consensusConfig
	consensusConfigMap, ok := args["config"].(map[string]interface{})
	if !ok {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::updateConsensus::config should be map.")
		return nil, EmptyConfig
	}
	consensusConfigBytes, err := json.Marshal(&consensusConfigMap)
	if err != nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::updateConsensus::unmarshal config err.", "err", err)
		return nil, EmptyConfig
	}
	return &base.ConsensusConfig{
		ConsensusName: consensusName,
		Config:        string(consensusConfigBytes),
		Index:         pc.stepConsensus.len(),
		StartHeight:   updateHeight + 1,
	}, nil
}

// updateConsensus 共识升级，更新原有共识列表，向可插拔共识列表插入新共识，并暂停原共识实例
// 该方法在trigger高度时被调用，此时共识version需要递增序列
func (pc *PluggableConsensusImpl) updateConsensus(contractCtx contractBase.KContext) (*contractBase.Response, error) {
	// 解析用户合约信息，包括待升级名称name、trigger高度height和待升级配置config
	cfg, err := pc.proposalArgsUnmarshal(contractCtx.Args())
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable CommonConsensus::updateConsensus::proposalArgsUnmarshal error", "error", err)
		return base.NewContractErrResponse(err.Error()), err
	}

	// 不允许升级为 pow 类共识
	if cfg.ConsensusName == "pow" {
		pc.ctx.XLog.Warn("Pluggable CommonConsensus::updateConsensus can not be pow")
		return base.NewContractErrResponse("Pluggable CommonConsensus::updateConsensus target can not be pow"),
			errors.New("updateConsensus target can not be pow")
	}

	// 当前共识如果是pow类共识，不允许升级
	if cur := pc.stepConsensus.tail(); cur != nil {
		if curStatus, err := cur.GetConsensusStatus(); err != nil || curStatus.GetConsensusName() == "pow" {
			pc.ctx.XLog.Warn("Pluggable CommonConsensus::updateConsensus current consensus is pow, can not upgrade from pow", "err", err)
			return base.NewContractErrResponse("Pluggable CommonConsensus::updateConsensus current consensus is pow"),
				errors.New("updateConsensus can not upgrade from pow")
		}
	}

	// 更新合约存储
	pluggableConfig, _ := contractCtx.Get(contractBucket, []byte(consensusKey))
	c := map[int]base.ConsensusConfig{}
	// 尚未写入过任何值，此时需要先写入genesisConfig，即初始共识配置值
	if pluggableConfig == nil {
		consensusBuf, _ := pc.ctx.Ledger.GetConsensusConf()
		config := base.ConsensusConfig{}
		_ = json.Unmarshal(consensusBuf, &config)
		config.StartHeight = 1
		config.Index = 0
		c[0] = config
	} else {
		err = json.Unmarshal(pluggableConfig, &c)
		if err != nil {
			pc.ctx.XLog.Warn("Pluggable CommonConsensus::updateConsensus::unmarshal error", "error", err)
			return base.NewContractErrResponse(BuildConsensusError.Error()), BuildConsensusError
		}
	}

	// 检查生效高度
	if err := pc.checkConsensusHeight(cfg); err != nil {
		pc.ctx.GetLog().Error("Pluggable CommonConsensus::updateConsensus::check consensus height error")
		return base.NewContractErrResponse(err.Error()), err
	}

	// 检查新共识配置是否正确
	if err := checkConsensusVersion(c, cfg); err != nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::updateConsensus::wrong value, pls check your proposal file.", "error", err)
		return base.NewContractErrResponse(err.Error()), err
	}

	// 生成新的共识实例
	consensusItem, err := pc.makeConsensusItem(pc.ctx, c[len(c)-1])
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable CommonConsensus::updateConsensus::make consensu item error! Use old one.", "error", err.Error())
		return base.NewContractErrResponse(err.Error()), err
	}
	pc.ctx.XLog.Debug("Pluggable CommonConsensus::updateConsensus::make a new consensus item successfully during updating process.")

	newBytes, err := json.Marshal(c)
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable CommonConsensus::updateConsensus::marshal error", "error", err)
		return base.NewContractErrResponse(BuildConsensusError.Error()), BuildConsensusError
	}
	//合约存储持久化
	if err = contractCtx.Put(contractBucket, []byte(consensusKey), newBytes); err != nil {
		pc.ctx.XLog.Warn("Pluggable CommonConsensus::updateConsensus::refresh contract storage error", "error", err)
		return base.NewContractErrResponse(BuildConsensusError.Error()), BuildConsensusError
	}

	// 设置共识切换标志
	pc.stepConsensus.setSwitch(true)
	err = pc.stepConsensus.put(consensusItem)
	if err != nil {
		pc.ctx.XLog.Warn("Pluggable CommonConsensus::updateConsensus::put item into stepConsensus failed", "error", err)
		return base.NewContractErrResponse(BuildConsensusError.Error()), BuildConsensusError
	}
	pc.ctx.XLog.Debug("Pluggable CommonConsensus::updateConsensus::key has been modified.", "ConsensusMap", c)
	return base.NewContractOKResponse([]byte("ok")), nil
}

// CheckConsensusConfig 同名配置文件检查:
// 1. 同一个链的共识版本只能增加，不能升级到旧版本
// 2. 将合法的配置写到map中
func checkConsensusVersion(hisMap map[int]base.ConsensusConfig, cfg *base.ConsensusConfig) error {
	var err error
	var newConf configFilter
	if err = json.Unmarshal([]byte(cfg.Config), &newConf); err != nil {
		return errors.New("wrong parameter config")
	}
	newConfVersion, err := strconv.ParseInt(newConf.Version, 10, 64)
	if err != nil {
		return errors.New("wrong parameter version, version should an integer in string")
	}
	// 获取历史最近共识实例，初始状态下历史共识没有version字段，需手动添加
	var maxVersion int64
	for i := len(hisMap) - 1; i >= 0; i-- {
		configItem := hisMap[i]
		var tmpItem configFilter
		err := json.Unmarshal([]byte(configItem.Config), &tmpItem)
		if err != nil {
			return errors.New("unmarshal config error")
		}
		if tmpItem.Version == "" {
			tmpItem.Version = "0"
		}
		v, _ := strconv.ParseInt(tmpItem.Version, 10, 64)
		if maxVersion < v {
			maxVersion = v
		}
	}
	if maxVersion < newConfVersion {
		hisMap[len(hisMap)] = *cfg
		return nil
	}
	return ErrInvalidVersion
}

type configFilter struct {
	Version string `json:"version,omitempty"`
}

// checkConsensusHeight 检查区块高度，距离上次升级高度 > 20
func (pc *PluggableConsensusImpl) checkConsensusHeight(cfg *base.ConsensusConfig) error {
	con := pc.stepConsensus.tail()
	if con == nil {
		pc.ctx.GetLog().Warn("check consensus height error")
		return errors.New("check consensus height error")
	}
	conStatus, _ := con.GetConsensusStatus()
	if cfg.StartHeight-conStatus.GetConsensusBeginInfo() < 20 {
		pc.ctx.GetLog().Warn("check consensus height error, at least more than 20 block by last ")
		return errors.New("check consensus height error")
	}
	return nil
}

// CompeteMaster 矿工检查当前自己是否需要挖矿，需要账本当前最高的高度作为输入
func (pc *PluggableConsensusImpl) CompeteMaster(height int64) (bool, bool, error) {
	con, _ := pc.getCurrentConsensusItem(height)
	if con == nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::CompeteMaster::Cannot get consensus Instance.")
		return false, false, EmptyConsensusListErr
	}
	return con.CompeteMaster(height)
}

// CheckMinerMatch 调用具体实例的CheckMinerMatch()
func (pc *PluggableConsensusImpl) CheckMinerMatch(ctx xctx.Context, block ledger.BlockHandle) (bool, error) {
	con, _ := pc.getCurrentConsensusItem(block.GetHeight())
	if con == nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::CheckMinerMatch::tail consensus item is empty", "err", EmptyConsensusListErr)
		return false, EmptyConsensusListErr
	}
	return con.CheckMinerMatch(ctx, block)
}

// ProcessBeforeMiner 调用具体实例的ProcessBeforeMiner()
func (pc *PluggableConsensusImpl) ProcessBeforeMiner(height, timestamp int64) ([]byte, []byte, error) {
	con, _ := pc.getCurrentConsensusItem(height)
	if con == nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::ProcessBeforeMiner::tail consensus item is empty", "err", EmptyConsensusListErr)
		return nil, nil, EmptyConsensusListErr
	}
	return con.ProcessBeforeMiner(height, timestamp)
}

// CalculateBlock 矿工挖矿时共识需要做的工作, 如PoW时共识需要完成存在性证明
func (pc *PluggableConsensusImpl) CalculateBlock(block ledger.BlockHandle) error {
	con, _ := pc.getCurrentConsensusItem(block.GetHeight())
	if con == nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::CalculateBlock::tail consensus item is empty", "err", EmptyConsensusListErr)
		return EmptyConsensusListErr
	}
	return con.CalculateBlock(block)
}

// ProcessConfirmBlock 调用具体实例的ProcessConfirmBlock()
func (pc *PluggableConsensusImpl) ProcessConfirmBlock(block ledger.BlockHandle) error {
	con, _ := pc.getCurrentConsensusItem(block.GetHeight())
	if con == nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::ProcessConfirmBlock::tail consensus item is empty", "err", EmptyConsensusListErr)
		return EmptyConsensusListErr
	}
	return con.ProcessConfirmBlock(block)
}

// GetConsensusStatus 调用具体实例的GetConsensusStatus()
func (pc *PluggableConsensusImpl) GetConsensusStatus() (base.ConsensusStatus, error) {
	block := pc.ctx.Ledger.GetTipBlock()
	con, _ := pc.getCurrentConsensusItem(block.GetHeight() + 1)
	if con == nil {
		pc.ctx.XLog.Error("Pluggable CommonConsensus::GetConsensusStatus::tail consensus item is empty", "err", EmptyConsensusListErr)
		return nil, EmptyConsensusListErr
	}
	return con.GetConsensusStatus()
}

// SwitchConsensus 用于共识升级时切换共识实例
func (pc *PluggableConsensusImpl) SwitchConsensus(height int64) error {
	// 获取最新的共识实例
	con := pc.stepConsensus.tail()
	if con == nil {
		pc.ctx.XLog.Error("pluggable consensus SwitchConsensus stepConsensus.tail error")
		return errors.New("pluggable consensus SwitchConsensus stepConsensus.tail error")
	}

	// 获取最新实例的共识状态
	consensusStatus, err := con.GetConsensusStatus()
	if err != nil {
		pc.ctx.XLog.Error("pluggable consensus SwitchConsensus GetConsensusStatus failed", "error", err)
		return errors.New("pluggable consensus SwitchConsensus GetConsensusStatus failed")
	}
	pc.ctx.XLog.Debug("pluggable consensus SwitchConsensus", "block height", height,
		"current consensus start height", consensusStatus.GetConsensusBeginInfo(), "pc.stepConsensus.getSwitch()", pc.stepConsensus.getSwitch())

	if height >= consensusStatus.GetConsensusBeginInfo()-1 && pc.stepConsensus.getSwitch() {
		pc.ctx.XLog.Debug("pluggable consensus SwitchConsensus switch consensus is true")
		// 由于共识升级切换期间涉及到新老共识并存的问题，如果矿工已经打包更高的区块，那么可以启动新共识，关闭老共识
		preCon := pc.stepConsensus.preTail()
		if preCon != nil {
			_ = preCon.Stop()
			pc.ctx.XLog.Debug("pluggable consensus SwitchConsensus switch stop pre consensus success")
		}
		con := pc.stepConsensus.tail()
		if con == nil {
			pc.ctx.XLog.Error("pluggable consensus SwitchConsensus stepConsensus.tail error")
			return errors.New("pluggable consensus SwitchConsensus stepConsensus.tail error")
		}
		if err := con.Start(); err != nil {
			pc.ctx.XLog.Error("pluggable consensus SwitchConsensus start new consensus failed", "error", err)
			return errors.New("pluggable consensus SwitchConsensus start new consensus failed")
		}
		pc.ctx.XLog.Debug("pluggable consensus SwitchConsensus switch start new consensus success")
		// 关闭共识切换开关
		pc.stepConsensus.setSwitch(false)
	}
	return nil
}

func (pc *PluggableConsensusImpl) getCurrentConsensusItem(height int64) (base.CommonConsensus, error) {
	con := pc.stepConsensus.tail()
	if con == nil {
		pc.ctx.XLog.Error("pluggable consensus stepConsensus.tail error")
		return nil, errors.New("pluggable consensus stepConsensus.tail error")
	}

	// 获取最新实例的共识状态
	consensusStatus, err := con.GetConsensusStatus()
	if err != nil {
		pc.ctx.XLog.Error("pluggable consensus GetConsensusStatus failed", "error", err)
		return nil, errors.New("pluggable consensus GetConsensusStatus failed")
	}

	// 判断当前区块的高度是否>=最新实例共识起始高度
	if height >= consensusStatus.GetConsensusBeginInfo() {
		return con, nil
	}

	pc.ctx.XLog.Debug("pluggable consensus start use pre consensus", "height", height)
	preCon := pc.stepConsensus.preTail()
	if preCon == nil {
		pc.ctx.XLog.Error("pluggable consensus stepConsensus.preTail error")
		return nil, errors.New("pluggable consensus stepConsensus.preTail error")
	}
	return preCon, nil
}
