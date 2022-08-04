package kernel

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	contractBase "github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/contract/bridge"
	"github.com/wooyang2018/corechain/contract/sandbox"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/permission/base"
)

const (
	contractConfigName = "contract.yaml"
)

type managerImpl struct {
	core      contractBase.ChainCore
	xbridge   *bridge.XBridge
	kregistry registryImpl
}

func loadConfig(fname string) (*contractBase.ContractConfig, error) {
	viperObj := viper.New()
	viperObj.SetConfigFile(fname)
	err := viperObj.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("read config failed.path:%s,err:%v", fname, err)
	}

	cfg := contractBase.DefaultContractConfig()
	if err = viperObj.Unmarshal(&cfg, func(config *mapstructure.DecoderConfig) {
		config.TagName = "yaml"
	}); err != nil {
		return nil, fmt.Errorf("unmatshal config failed.path:%s,err:%v", fname, err)
	}
	return cfg, nil
}

func newManagerImpl(cfg *contractBase.ManagerConfig) (contractBase.Manager, error) {
	if cfg.Basedir == "" || !filepath.IsAbs(cfg.Basedir) {
		return nil, errors.New("base dir of contract manager must be absolute")
	}
	if cfg.BCName == "" {
		return nil, errors.New("empty chain name when init contract manager")
	}
	if cfg.Core == nil {
		return nil, errors.New("nil chain core when init contract manager")
	}
	if cfg.XMReader == nil {
		return nil, errors.New("nil xmodel reader when init contract manager")
	}
	if cfg.EnvConf == nil && cfg.Config == nil {
		return nil, errors.New("nil contract config when init contract manager")
	}
	var xcfg *contractBase.ContractConfig
	if cfg.EnvConf == nil {
		xcfg = cfg.Config
	} else {
		var err error
		xcfg, err = loadConfig(cfg.EnvConf.GenConfFilePath(contractConfigName))
		if err != nil {
			return nil, fmt.Errorf("error while load contract config:%s", err)
		}
	}

	m := &managerImpl{
		core: cfg.Core,
	}
	var logDriver logger.Logger
	if cfg.Config != nil {
		logDriver = cfg.Config.LogDriver
	}
	xbridge, err := bridge.New(&bridge.XBridgeConfig{
		Basedir: cfg.Basedir,
		VMConfigs: map[bridge.ContractType]bridge.VMConfig{
			bridge.TypeWasm:   &xcfg.Wasm,
			bridge.TypeNative: &xcfg.Native,
			bridge.TypeEvm:    &xcfg.EVM,
			bridge.TypeKernel: &contractBase.XkernelConfig{
				Driver:   xcfg.Xkernel.Driver,
				Enable:   xcfg.Xkernel.Enable,
				Registry: &m.kregistry,
			},
		},
		Config:    *xcfg,
		XModel:    cfg.XMReader,
		Core:      cfg.Core,
		LogDriver: logDriver,
	})
	if err != nil {
		return nil, err
	}
	m.xbridge = xbridge
	registry := &m.kregistry
	registry.RegisterKernMethod("$contract", "deployContract", m.deployContract)
	registry.RegisterKernMethod("$contract", "upgradeContract", m.upgradeContract)
	registry.RegisterShortcut("Deploy", "$contract", "deployContract")
	registry.RegisterShortcut("Upgrade", "$contract", "upgradeContract")
	return m, nil
}

func (m *managerImpl) NewContext(cfg *contractBase.ContextConfig) (contractBase.VMContext, error) {
	return m.xbridge.NewContext(cfg)
}

func (m *managerImpl) NewStateSandbox(cfg *contractBase.SandboxConfig) (contractBase.StateSandbox, error) {
	return sandbox.NewXModelCache(cfg), nil
}

func (m *managerImpl) GetKernRegistry() contractBase.KernRegistry {
	return &m.kregistry
}

func (m *managerImpl) deployContract(ctx contractBase.KContext) (*contractBase.Response, error) {
	// check if account exist
	accountName := ctx.Args()["account_name"]
	contractName := ctx.Args()["contract_name"]
	if accountName == nil || contractName == nil {
		return nil, errors.New("invoke DeployMethod error, account name or contract name is nil")
	}
	// check if contractName is ok
	if err := contractBase.ValidContractName(string(contractName)); err != nil {
		return nil, fmt.Errorf("deploy failed, contract `%s` contains illegal character, error: %s", contractName, err)
	}
	_, err := ctx.Get(base.GetAccountBucket(), accountName)
	if err != nil {
		return nil, fmt.Errorf("get account `%s` error: %s", accountName, err)
	}

	resp, limit, err := m.xbridge.DeployContract(ctx)
	if err != nil {
		return nil, err
	}
	ctx.AddResourceUsed(limit)

	// key: contract, value: account
	err = ctx.Put(base.GetContract2AccountBucket(), contractName, accountName)
	if err != nil {
		return nil, err
	}
	key := base.MakeAccountContractKey(string(accountName), string(contractName))
	err = ctx.Put(base.GetAccount2ContractBucket(), []byte(key), []byte(base.GetAccountContractValue()))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *managerImpl) upgradeContract(ctx contractBase.KContext) (*contractBase.Response, error) {
	contractName := ctx.Args()["contract_name"]
	if contractName == nil {
		return nil, errors.New("invoke Upgrade error, contract name is nil")
	}

	err := m.core.VerifyContractOwnerPermission(string(contractName), ctx.AuthRequire())
	if err != nil {
		return nil, err
	}

	resp, limit, err := m.xbridge.UpgradeContract(ctx)
	if err != nil {
		return nil, err
	}
	ctx.AddResourceUsed(limit)
	return resp, nil
}

func init() {
	contractBase.Register("default", newManagerImpl)
}
