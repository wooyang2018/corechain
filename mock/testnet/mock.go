package testnet

import (
	"path/filepath"

	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/network"
	nctx "github.com/wooyang2018/corechain/network/context"
)

func GetMockEnvConf(paths ...string) (*xconf.EnvConf, error) {
	path := "conf/env.yaml"
	if len(paths) > 0 {
		path = paths[0]
	}

	dir := utils.GetCurFileDir()
	econfPath := filepath.Join(dir, path)
	econf, err := xconf.LoadEnvConf(econfPath)
	if err != nil {
		return nil, err
	}

	return econf, nil
}

func NewFakeP2P(node string, module ...string) (network.Network, *nctx.NetCtx, error) {
	ecfg, _ := GetMockEnvConf(node + "/conf/env.yaml")
	logger.InitMLog(ecfg.GenConfFilePath(ecfg.LogConf), ecfg.GenDirAbsPath(ecfg.LogDir))
	if module != nil && len(module) == 1 {
		ecfg.NetConf = module[0] + ".yaml"
	}
	ctx, _ := nctx.NewNetCtx(ecfg)
	netObj, err := network.NewNetwork(ctx)
	if err != nil {
		return nil, nil, err
	}

	return netObj, ctx, nil
}
