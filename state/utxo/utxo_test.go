package utxo_test

import (
	"math/big"
	"os"
	"testing"
	"time"

	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/ledger"
	lctx "github.com/wooyang2018/corechain/ledger/context"
	ltx "github.com/wooyang2018/corechain/ledger/tx"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/state"
	"github.com/wooyang2018/corechain/state/context"
	"github.com/wooyang2018/corechain/state/meta"
	"github.com/wooyang2018/corechain/state/utxo"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
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

func TestBasicFunc(t *testing.T) {
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
                        "quota" : "100000000"
                },
				{
                        "address" : "` + AliceAddress + `",
                        "quota" : "200000000"
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

	sctx, err := context.NewStateCtx(econf, "corechain", mledger, crypt)
	if err != nil {
		t.Fatal(err)
	}

	sctx.EnvCfg.ChainDir = workspace
	stateHandle, _ := state.NewState(sctx)

	// test for HasTx
	exist, _ := stateHandle.HasTx(tx.Txid)
	t.Log("Has tx ", tx.Txid, exist)
	err = stateHandle.DoTx(tx)
	if err != nil {
		t.Log("coinbase do tx error ", err.Error())
	}

	playErr := stateHandle.Play(block.Blockid)
	if playErr != nil {
		t.Fatal(playErr)
	}

	metaHandle, err := meta.NewMeta(sctx, stateHandle.GetLDB())
	if err != nil {
		t.Fatal(err)
	}
	utxoHandle, err := utxo.NewUtxo(sctx, metaHandle, stateHandle.GetLDB())
	if err != nil {
		t.Fatal(err)
	}
	balance, err := utxoHandle.GetBalance(BobAddress)
	utxoHandle.AddBalance([]byte(BobAddress), big.NewInt(10000000))
	balance, err = utxoHandle.GetBalance(BobAddress)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("get balance", balance.String())

	tx1 := &protos.Transaction{}
	tx1.Nonce = "nonce"
	tx1.Timestamp = time.Now().UnixNano()
	tx1.Desc = []byte("desc")
	tx1.Version = 1
	tx1.AuthRequire = append(tx.AuthRequire, BobAddress)
	tx1.Initiator = BobAddress
	tx1.Coinbase = false
	totalNeed := big.NewInt(0) // 需要支付的总额
	amountBig := big.NewInt(0)
	amountBig.SetString("10", 10) // 10进制转换大整数
	totalNeed.Add(totalNeed, amountBig)
	totalNeed.Add(totalNeed, amountBig)
	txOutput := &protos.TxOutput{}
	txOutput.ToAddr = []byte(AliceAddress)
	txOutput.Amount = amountBig.Bytes()
	txOutput.FrozenHeight = 0
	tx.TxOutputs = append(tx.TxOutputs, txOutput)
	txInputs, _, utxoTotal, err := utxoHandle.SelectUtxos(BobAddress, totalNeed, true, false)
	if err != nil {
		t.Fatal(err)
	}
	tx1.TxInputs = txInputs
	if utxoTotal.Cmp(totalNeed) > 0 {
		delta := utxoTotal.Sub(utxoTotal, totalNeed)
		txOutput := &protos.TxOutput{}
		txOutput.ToAddr = []byte(BobAddress) // 收款人就是汇款人自己
		txOutput.Amount = delta.Bytes()
		tx1.TxOutputs = append(tx1.TxOutputs, txOutput)
	}
	err = utxoHandle.CheckInputEqualOutput(tx1, nil)
	if err != nil {
		t.Log(err)
	}
	txInputs, _, utxoTotal, err = utxoHandle.SelectUtxosBySize(BobAddress, true, false)
	if err != nil {
		t.Fatal(err)
	}
	total := utxoHandle.GetTotal()
	t.Log("total", total.String())
	utxoHandle.UpdateUtxoTotal(big.NewInt(200), stateHandle.NewBatch(), true)
	utxoHandle.UpdateUtxoTotal(big.NewInt(100), stateHandle.NewBatch(), false)
	total = utxoHandle.GetTotal()
	t.Log("total", total.String())

	txInputs, _, utxoTotal, err = utxoHandle.SelectUtxosBySize(BobAddress, false, false)
	if err != nil {
		t.Fatal(err)
	}

	accounts, err := utxoHandle.QueryAccountContainAK(BobAddress)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Log("accounts", accounts)
	}

	utxoHandle.SubBalance([]byte(BobAddress), big.NewInt(100))

	_, err = utxoHandle.QueryContractStatData()
	if err != nil {
		t.Fatal(err)
	}
	keys := utxo.MakeUtxoKey([]byte("U_TEST"), "1000")
	t.Log("keys", keys)

	cs, err := utxoHandle.GetAccountContracts("XC1111111111111111@xuper")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("contracts", cs)
	record, err := utxoHandle.QueryUtxoRecord("XC1111111111111111@xuper", 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("records", record)
}
