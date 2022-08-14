package model

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/ledger"
	ledgerBase "github.com/wooyang2018/corechain/ledger/base"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state/base"
	"github.com/wooyang2018/corechain/state/txhash"
	"github.com/wooyang2018/corechain/storage/leveldb"

	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

const (
	BobAddress = "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"
)

func TestSnapshotGet(t *testing.T) {
	//--------- 初始化工作 ----------
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

	t1 := &protos.Transaction{}
	t1.TxOutputs = append(t1.TxOutputs, &protos.TxOutput{Amount: []byte("888"), ToAddr: []byte(BobAddress)})
	t1.Coinbase = true
	t1.Desc = []byte(`{"maxblocksize" : "128"}`)
	t1.Txid, _ = txhash.MakeTxID(t1)
	block, err := mledger.FormatRootBlock([]*protos.Transaction{t1})
	if err != nil {
		t.Fatal(err)
	}

	confirmStatus := mledger.ConfirmBlock(block, true)
	if !confirmStatus.Succ {
		t.Fatal(fmt.Errorf("confirm block fail"))
	}

	crypt, err := client.CreateCryptoClient(client.CryptoTypeDefault)
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
	xmod, err := NewXModel(sctx, ldb)
	if err != nil {
		t.Fatal(err)
	}

	//------------- 测试Snapshot  ------------
	blkId, err := mledger.QueryBlockByHeight(0)
	if err != nil {
		t.Fatal(err)
	}

	xmsp, err := xmod.CreateSnapshot(blkId.Blockid)
	if err != nil {
		t.Log(err)
	}

	vData, err := xmsp.Get("proftestc", []byte("key_1"))
	if err != nil {
		t.Log(err)
	}

	fmt.Println(vData)
	fmt.Println(hex.EncodeToString(vData.RefTxid))

	mledger.Close()
}
