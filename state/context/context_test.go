package context

import (
	"os"
	"testing"

	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/ledger"
	lctx "github.com/wooyang2018/corechain/ledger/context"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

func TestNewNetCtx(t *testing.T) {
	workspace := mock.GetTempDirPath()
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)

	ecfg, err := mock.GetMockEnvConf()
	logger.InitMLog(ecfg.GenConfFilePath(ecfg.LogConf), ecfg.GenDirAbsPath(ecfg.LogDir))
	if err != nil {
		t.Fatal(err)
	}

	lctx, err := lctx.NewLedgerCtx(ecfg, "corechain")
	if err != nil {
		t.Fatal(err)
	}
	lctx.EnvCfg.ChainDir = workspace

	genesisConf := []byte(`
		{
    "version": "1",
    "predistribution": [
        {
            "address": "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
            "quota": "100000000000000000000"
        }
    ],
    "maxblocksize": "16",
    "award": "1000000",
    "decimals": "8",
    "award_decay": {
        "height_gap": 31536000,
        "ratio": 1
    },
    "gas_price": {
        "cpu_rate": 1000,
        "mem_rate": 1000000,
        "disk_rate": 1,
        "xfee_rate": 1
    },
    "new_account_resource_amount": 1000,
    "genesis_consensus": {
        "name": "single",
        "mock": {
            "miner": "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY",
            "period": 3000
        }
    }
}
    `)
	ledgerIns, err := ledger.CreateLedger(lctx, genesisConf)
	if err != nil {
		t.Fatal(err)
	}
	gcc, err := cryptoClient.CreateCryptoClient("gm")
	if err != nil {
		t.Errorf("gen crypto cryptoClient fail.err:%v", err)
	}
	sctx, err := NewStateCtx(ecfg, "corechain", ledgerIns, gcc)
	if err != nil {
		t.Fatal(err)
	}
	sctx.XLog.Debug("test NewNetCtx succ", "sctx", sctx)

	isInit := sctx.IsInit()
	t.Log("is init", isInit)
}
