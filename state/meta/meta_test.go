package meta

import (
	"os"
	"path/filepath"
	"testing"

	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/ledger"
	ledgerBase "github.com/wooyang2018/corechain/ledger/base"
	ltx "github.com/wooyang2018/corechain/ledger/tx"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state/base"
	"github.com/wooyang2018/corechain/storage/leveldb"
)

// base test data
const (
	BobAddress   = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	AliceAddress = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"
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

func TestMeta(t *testing.T) {
	//-------------- 初始化 --------------
	workspace := mock.GetTempDirPath()
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	econf, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	logger.InitMLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))

	lctx, err := ledgerBase.NewLedgerCtx(econf, "corechain")
	if err != nil {
		t.Fatal(err)
	}
	lctx.EnvCfg.ChainDir = workspace

	mledger, err := ledger.CreateLedger(lctx, GenesisConf)
	if err != nil {
		t.Fatal(err)
	}
	//创建链的时候分配财富
	tx, err := ltx.GenerateRootTx([]byte(`
       {
        "version" : "1"
        , "consensus" : {
                "miner" : "0x00000000000"
        }
        , "predistribution":[
                {
                        "address" : "` + BobAddress + `",
                        "quota" : "10000000"
                },
				{
                        "address" : "` + AliceAddress + `",
                        "quota" : "20000000"
                }

        ]
        , "maxblocksize" : "128"
        , "period" : "5000"
        , "award" : "1000"
		} 
    `))
	if err != nil {
		t.Fatal(err)
	}

	block, _ := mledger.FormatRootBlock([]*protos.Transaction{tx})
	t.Logf("blockid %x", block.Blockid)
	confirmStatus := mledger.ConfirmBlock(block, true)
	if !confirmStatus.Succ {
		t.Fatal("confirm block fail")
	}

	crypt, err := cryptoClient.CreateCryptoClient(cryptoClient.CryptoTypeDefault)
	if err != nil {
		t.Fatal(err)
	}

	sctx, err := base.NewStateCtx(econf, "corechain", mledger, crypt)
	if err != nil {
		t.Fatal(err)
	}
	sctx.EnvCfg.ChainDir = workspace
	storePath := sctx.EnvCfg.GenDataAbsPath(sctx.EnvCfg.ChainDir)
	storePath = filepath.Join(storePath, sctx.BCName)
	stateDBPath := filepath.Join(storePath, ledgerBase.StateStrgDirName)
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

	//-------------- 测试Meta --------------
	metaHadler, err := NewMeta(sctx, ldb)
	if err != nil {
		t.Fatal(err)
	}
	maxBlockSize := metaHadler.GetMaxBlockSize()
	t.Log("get succ", "maxBlockSize", maxBlockSize)
	slideWindow := metaHadler.GetIrreversibleSlideWindow()
	t.Log("get succ", "slideWindow", slideWindow)
	forbid := metaHadler.GetForbiddenContract()
	t.Log("get succ", "forbidContract", forbid)
	gchain := metaHadler.GetGroupChainContract()
	t.Log("get succ", "groupChainContract", gchain)
	gasPrice := metaHadler.GetGasPrice()
	t.Log("get succ", "gasPrice", gasPrice)
	bHeight := metaHadler.GetIrreversibleBlockHeight()
	t.Log("get succ", "blockHeight", bHeight)
	amount := metaHadler.GetNewAccountResourceAmount()
	t.Log("get succ", "newAccountResourceAmount", amount)
	contracts := metaHadler.GetReservedContracts()
	if len(contracts) == 0 {
		t.Log("empty reserved contracts")
	}
	_, err = metaHadler.MaxTxSizePerBlock()
	if err != nil {
		t.Fatal(err)
	}

	batch := ldb.NewBatch()
	err = metaHadler.UpdateIrreversibleBlockHeight(2, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateNextIrreversibleBlockHeightForPrune(3, 2, 1, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateNextIrreversibleBlockHeight(3, 3, 1, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateIrreversibleSlideWindow(2, batch)
	if err != nil {
		t.Fatal(err)
	}
	gasPrice = &protos.GasPrice{
		CpuRate:  100,
		MemRate:  100000,
		DiskRate: 1,
		XfeeRate: 1,
	}
	err = metaHadler.UpdateGasPrice(gasPrice, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateNewAccountResourceAmount(500, batch)
	if err != nil {
		t.Fatal(err)
	}
	err = metaHadler.UpdateMaxBlockSize(64, batch)
	if err != nil {
		t.Fatal(err)
	}
	reqs := make([]*protos.InvokeRequest, 0, 1)
	request := &protos.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: "identity",
		MethodName:   "verify",
		Args:         map[string][]byte{},
	}
	reqs = append(reqs, request)
	err = metaHadler.UpdateReservedContracts(reqs, batch)
	if err != nil {
		t.Fatal(err)
	}
	upForbidRequest := &protos.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: "forbidden",
		MethodName:   "get",
		Args:         map[string][]byte{},
	}
	err = metaHadler.UpdateForbiddenContract(upForbidRequest, batch)
	if err != nil {
		t.Fatal(err)
	}
}
