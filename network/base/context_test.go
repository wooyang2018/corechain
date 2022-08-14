package base

import (
	"testing"

	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
)

func TestNewNetCtx(t *testing.T) {
	envCfg, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	logger.InitMLog(envCfg.GenConfFilePath(envCfg.LogConf), envCfg.GenDirAbsPath(envCfg.LogDir))
	octx, err := NewNetCtx(envCfg)
	if err != nil {
		t.Fatal(err)
	}
	octx.XLog.Debug("test NewNetCtx succ", "cfg", octx.P2PConf)
}
