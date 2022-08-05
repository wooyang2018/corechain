package tx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wooyang2018/corechain/storage/leveldb"

	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/ledger"
	lctx "github.com/wooyang2018/corechain/ledger/context"
	"github.com/wooyang2018/corechain/ledger/def"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/state/base"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

var GenesisConf = []byte(`
		{
    "version": "1",
    "predistribution": [
        {
            "address": "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
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
            "miner": "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
            "period": 3000
        }
    }
}
    `)

func TestTx(t *testing.T) {
	_, minerErr := GenerateAwardTx("miner-1", "1000", []byte("award"))
	if minerErr != nil {
		t.Fatal(minerErr)
	}
	_, etxErr := GenerateEmptyTx([]byte("empty"))
	if etxErr != nil {
		t.Fatal(minerErr)
	}
	_, rtxErr := GenerateRootTx(GenesisConf)
	if rtxErr != nil {
		t.Fatal(rtxErr)
	}

	workspace := mock.GetTempDirPath()
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	econf, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	econf.ChainDir = workspace
	logger.InitMLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))

	lctx, err := lctx.NewLedgerCtx(econf, "corechain")
	if err != nil {
		t.Fatal(err)
	}

	mledger, err := ledger.CreateLedger(lctx, GenesisConf)
	if err != nil {
		t.Fatal(err)
	}

	crypt, err := cryptoClient.CreateCryptoClient(cryptoClient.CryptoTypeDefault)
	if err != nil {
		t.Fatal(err)
	}

	sctx, err := base.NewStateCtx(econf, "corechain", mledger, crypt)
	if err != nil {
		t.Fatal(err)
	}

	storePath := sctx.EnvCfg.GenDataAbsPath(sctx.EnvCfg.ChainDir)
	storePath = filepath.Join(storePath, sctx.BCName)
	stateDBPath := filepath.Join(storePath, def.StateStrgDirName)
	kvParam := &leveldb.KVParameter{
		DBPath:                stateDBPath,
		KVEngineType:          sctx.LedgerCfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            sctx.LedgerCfg.OtherPaths,
		StorageType:           sctx.LedgerCfg.StorageType,
	}
	ldb, err := leveldb.CreateKVInstance(kvParam)
	if err != nil {
		t.Fatal(err)
	}
	txHandle, err := NewTxHandler(sctx, ldb)
	if err != nil {
		t.Fatal(err)
	}
	err = txHandle.LoadUnconfirmedTxFromDisk()
	if err != nil {
		t.Fatal(err)
	}
	unConfirmedTx, err := txHandle.GetUnconfirmedTx(false, 0)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("unconfirmed tx len:", len(unConfirmedTx))
	}
	txHandle.SetMaxConfirmedDelay(500)
	txs, txDelay, err := txHandle.SortUnconfirmedTx(0)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("sort txs", "txMap", txs, txDelay)
	}
}
