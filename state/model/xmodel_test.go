package model

import (
	"fmt"
	"github.com/wooyang2018/corechain/storage/leveldb"
	"os"
	"path/filepath"
	"testing"

	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/ledger"
	lctx "github.com/wooyang2018/corechain/ledger/context"
	"github.com/wooyang2018/corechain/ledger/def"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state/context"
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

func TestBaiscFunc(t *testing.T) {
	workspace := mock.GetTempDirPath()
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	econf, err := mock.GetMockEnvConf()
	if err != nil {
		t.Fatal(err)
	}
	logger.InitMLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))

	ctx, err := lctx.NewLedgerCtx(econf, "corechain")
	if err != nil {
		t.Fatal(err)
	}
	ctx.EnvCfg.ChainDir = workspace
	mledger, err := ledger.CreateLedger(ctx, GenesisConf)
	if err != nil {
		t.Fatal(err)
	}

	crypt, err := cryptoClient.CreateCryptoClient(cryptoClient.CryptoTypeDefault)
	if err != nil {
		t.Fatal(err)
	}

	sctx, err := context.NewStateCtx(econf, "corechain", mledger, crypt)
	if err != nil {
		t.Fatal(err)
	}
	sctx.EnvCfg.ChainDir = workspace

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
	xModel, err := NewXModel(sctx, ldb)
	if err != nil {
		t.Fatal(err)
	}
	verData, err := xModel.Get("bucket1", []byte("hello"))
	if !IsEmptyVersionedData(verData) {
		t.Fatal("unexpected")
	}
	tx1 := &protos.Transaction{
		Txid: []byte("Tx1"),
		TxInputsExt: []*protos.TxInputExt{
			&protos.TxInputExt{
				Bucket: "bucket1",
				Key:    []byte("hello"),
			},
		},
		TxOutputsExt: []*protos.TxOutputExt{
			&protos.TxOutputExt{
				Bucket: "bucket1",
				Key:    []byte("hello"),
				Value:  []byte("you are the best!"),
			},
		},
	}
	batch := ldb.NewBatch()
	err = xModel.DoTx(tx1, batch)
	if err != nil {
		t.Fatal(err)
	}
	saveUnconfirmTx(tx1, batch)
	err = batch.Write()
	if err != nil {
		t.Fatal(err)
	}
	verData, err = xModel.Get("bucket1", []byte("hello"))
	if GetVersion(verData) != fmt.Sprintf("%x_%d", "Tx1", 0) {
		t.Fatal("unexpected", GetVersion(verData))
	}
	tx2 := &protos.Transaction{
		Txid: []byte("Tx2"),
		TxInputsExt: []*protos.TxInputExt{
			&protos.TxInputExt{
				Bucket:    "bucket1",
				Key:       []byte("hello"),
				RefTxid:   []byte("Tx1"),
				RefOffset: 0,
			},
			&protos.TxInputExt{
				Bucket: "bucket1",
				Key:    []byte("world"),
			},
		},
		TxOutputsExt: []*protos.TxOutputExt{
			&protos.TxOutputExt{
				Bucket: "bucket1",
				Key:    []byte("hello"),
				Value:  []byte("\x00"),
			},
			&protos.TxOutputExt{
				Bucket: "bucket1",
				Key:    []byte("world"),
				Value:  []byte("world is full of love!"),
			},
		},
	}
	_, err = ParseContractUtxoInputs(tx2)
	if err != nil {
		t.Fatal(err)
	}
	prefix := GenWriteKeyWithPrefix(tx2.TxOutputsExt[0])
	t.Log("gen prefix succ", "prefix", prefix)
	batch2 := ldb.NewBatch()
	err = xModel.DoTx(tx2, batch2)
	if err != nil {
		t.Fatal(err)
	}
	saveUnconfirmTx(tx2, batch2)
	err = batch2.Write()
	if err != nil {
		t.Fatal(err)
	}
	verData, err = xModel.Get("bucket1", []byte("hello"))
	if GetVersion(verData) != fmt.Sprintf("%x_%d", "Tx2", 0) {
		t.Fatal("unexpected", GetVersion(verData))
	}
	iter, err := xModel.Select("bucket1", []byte(""), []byte("\xff"))
	defer iter.Close()
	validKvCount := 0
	for iter.Next() {
		t.Logf("iter:  data %v, key: %s\n", iter.Value(), iter.Key())
		validKvCount++
	}
	if validKvCount != 1 {
		t.Fatal("unexpected", validKvCount)
	}
	_, isConfiremd, err := xModel.QueryTx(tx2.Txid)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("query succ", "isConfirmed", isConfiremd)
	}
	xModel.CleanCache()
	xModel.BucketCacheDelete("bucket1", GetVersion(verData))
	_, err = xModel.QueryBlock([]byte("123"))
	if err != nil {
		t.Log(err)
	}

	vData, err := xModel.GetFromLedger(tx2.TxInputsExt[0])
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log(vData)
	}
	version := MakeVersion(tx2.Txid, 0)
	txid := GetTxidFromVersion(version)
	t.Log("txid", txid)
	vDatas := make([]*ledger.VersionedData, 0, 1)
	vDatas = append(vDatas, vData)
	GetTxInputs(vDatas)

	pds := []*ledger.PureData{
		&ledger.PureData{
			Bucket: "bucket1",
			Key:    []byte("key1"),
			Value:  []byte("value1"),
		},
	}
	GetTxOutputs(pds)
	vData, exists, err := xModel.GetWithTxStatus("bucket1", []byte("hello"))
	if err != nil {
		t.Log(err)
	} else {
		t.Log("get txStatus succ", "data", vData, "exist", exists)
	}

	err = xModel.UndoTx(tx1, batch)
	if err != nil {
		t.Fatal(err)
	}

	mledger.Close()
}
