package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	protos2 "github.com/wooyang2018/corechain/example/protos"
	scom "github.com/wooyang2018/corechain/example/service/common"
	mock "github.com/wooyang2018/corechain/mock/config"

	xconf "github.com/wooyang2018/corechain/common/config"
	// import要使用的内核核心组件驱动
	_ "github.com/wooyang2018/corechain/consensus/pow"
	_ "github.com/wooyang2018/corechain/consensus/single"
	_ "github.com/wooyang2018/corechain/consensus/xpoa"
	_ "github.com/wooyang2018/corechain/consensus/xpos"
	_ "github.com/wooyang2018/corechain/contract/evm"
	_ "github.com/wooyang2018/corechain/contract/kernel"
	_ "github.com/wooyang2018/corechain/contract/native"
	_ "github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/engines"
	engineBase "github.com/wooyang2018/corechain/engines/base"
	ledgerBase "github.com/wooyang2018/corechain/ledger/base"
	ltx "github.com/wooyang2018/corechain/ledger/tx"
	"github.com/wooyang2018/corechain/logger"
	_ "github.com/wooyang2018/corechain/network/p2pv1"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

var (
	address   = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"
	publickey = "{\"Curvname\":\"P-256\",\"X\":74695617477160058757747208220371236837474210247114418775262229497812962582435,\"Y\":51348715319124770392993866417088542497927816017012182211244120852620959209571}"
)

func TestEndorserCall(t *testing.T) {
	workspace, dirErr := ioutil.TempDir("/tmp", "")
	if dirErr != nil {
		t.Fatal(dirErr)
	}
	os.RemoveAll(workspace)
	defer os.RemoveAll(workspace)
	conf, _ := mock.GetMockEnvConf()
	defer RemoveLedger(conf)

	engine, err := MockEngine()
	if err != nil {
		t.Fatal(err)
	}
	log, _ := logger.NewLogger("", scom.SubModName)
	rpcServ := NewRpcServ(engine, log)

	endor := NewDefaultXEndorser(rpcServ, engine)
	awardTx, err := ltx.GenerateAwardTx("miner", "1000", []byte("award"))

	txStatus := &protos2.TxStatus{
		Bcname: "xuper",
		Tx:     scom.TxToXchain(awardTx),
	}
	requestData, err := json.Marshal(txStatus)
	if err != nil {
		fmt.Printf("json encode txStatus failed: %v", err)
		t.Fatal(err)
	}
	ctx := context.TODO()
	req := &protos2.EndorserRequest{
		RequestName: "ComplianceCheck",
		BcName:      "xuper",
		Fee:         nil,
		RequestData: requestData,
	}
	resp, err := endor.EndorserCall(ctx, req)
	if err != nil {
		t.Log(err)
	}
	t.Log(resp)
	invokeReq := make([]*protos2.InvokeRequest, 0)
	invoke := &protos2.InvokeRequest{
		ModuleName:   "wasm",
		ContractName: "counter",
		MethodName:   "increase",
		Args:         map[string][]byte{"key": []byte("test")},
	}
	invokeReq = append(invokeReq, invoke)
	preq := &protos2.PreExecWithSelectUTXORequest{
		Bcname:      "xuper",
		Address:     address,
		TotalAmount: 100,
		SignInfo: &protos2.SignatureInfo{
			PublicKey: publickey,
			Sign:      []byte("sign"),
		},
		NeedLock: false,
		Request: &protos2.InvokeRPCRequest{
			Bcname:      "xuper",
			Requests:    invokeReq,
			Initiator:   address,
			AuthRequire: []string{address},
		},
	}

	reqJSON, _ := json.Marshal(preq)
	xreq := &protos2.EndorserRequest{
		RequestName: "PreExecWithFee",
		BcName:      "xuper",
		Fee:         nil,
		RequestData: reqJSON,
	}
	resp, err = endor.EndorserCall(ctx, xreq)
	if err != nil {
		//pass
		t.Log(err)
	}
	t.Log(resp)
	qtxTxStatus := &protos2.TxStatus{
		Bcname: "xuper",
		Txid:   []byte("70c64d6cb9b5647048d067c6775575fc52e3c51c6425cec3881d8564ad8e887c"),
	}
	requestData, err = json.Marshal(qtxTxStatus)
	if err != nil {
		fmt.Printf("json encode txStatus failed: %v", err)
		t.Fatal(err)
	}
	req = &protos2.EndorserRequest{
		RequestName: "TxQuery",
		BcName:      "xuper",
		RequestData: requestData,
	}
	resp, err = endor.EndorserCall(ctx, req)
	if err != nil {
		t.Log(err)
	}
	t.Log(resp)
}

func MockEngine() (engineBase.Engine, error) {
	conf, err := mock.GetMockEnvConf()
	if err != nil {
		return nil, fmt.Errorf("new env conf error: %v", err)
	}

	RemoveLedger(conf)
	if err = CreateLedger(conf); err != nil {
		return nil, err
	}

	engine := engines.NewEngine()
	if err := engine.Init(conf); err != nil {
		return nil, fmt.Errorf("init engine error: %v", err)
	}

	eng, err := engines.EngineConvert(engine)
	if err != nil {
		return nil, fmt.Errorf("engine convert error: %v", err)
	}

	return eng, nil
}

func RemoveLedger(conf *xconf.EnvConf) error {
	path := conf.GenDataAbsPath("blockchain")
	if err := os.RemoveAll(path); err != nil {
		log.Printf("remove ledger failed.err:%v\n", err)
		return err
	}
	return nil
}

func CreateLedger(conf *xconf.EnvConf) error {
	mockConf, err := mock.GetMockEnvConf()
	if err != nil {
		return fmt.Errorf("new mock env conf error: %v", err)
	}

	genesisPath := mockConf.GenDataAbsPath("genesis/xuper.json")
	err = ledgerBase.CreateLedger("xuper", genesisPath, conf)
	if err != nil {
		log.Printf("create ledger failed.err:%v\n", err)
		return fmt.Errorf("create ledger failed")
	}
	return nil
}
