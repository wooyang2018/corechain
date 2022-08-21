package worker

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/ledger"
	ledgerBase "github.com/wooyang2018/corechain/ledger/base"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/storage"
	"github.com/wooyang2018/corechain/storage/leveldb"
)

const (
	testBcName   = "corecahin"
	testContract = "$" + testBcName
	testEvent    = "CreateBlockChain"
)

func newTx() []*protos.FilteredTransaction {
	var txs []*protos.FilteredTransaction
	txs = append(txs, &protos.FilteredTransaction{
		Txid: "txid_1",
		Events: []*protos.ContractEvent{
			{
				Contract: testContract,
				Name:     testEvent,
				Body:     []byte("hello1"),
			},
		},
	})
	return txs
}

func newTxs() []*protos.FilteredTransaction {
	txs := newTx()
	txs = append(txs, &protos.FilteredTransaction{
		Txid: "txid_2",
		Events: []*protos.ContractEvent{
			{
				Contract: testContract,
				Name:     testEvent,
				Body:     []byte("hello2"),
			},
		},
	})
	return txs
}

//newAsyncWorker 新建异步服务代理
func newAsyncWorker() (*AsyncWorkerImpl, error) {
	basedir := mock.GetAbsTempDirPath()
	lcfg, err := ledgerBase.LoadLedgerConf(mock.GetLedgerConfFilePath())
	if err != nil {
		return nil, err
	}

	kvParam := &leveldb.KVParameter{
		DBPath:                filepath.Join(basedir, "async_worker"),
		KVEngineType:          lcfg.KVEngineType,
		MemCacheSize:          ledger.MemCacheSize,
		FileHandlersCacheSize: ledger.FileHandlersCacheSize,
		OtherPaths:            lcfg.OtherPaths,
		StorageType:           lcfg.StorageType,
	}
	baseDB, err := leveldb.CreateKVInstance(kvParam)
	if err != nil {
		return nil, err
	}

	tmpFinishTable := storage.NewTable(baseDB, FinishTablePrefix)

	// log实例
	mock.InitFakeLogger()
	tmpLog, _ := logger.NewLogger("", "asyncworker")

	aw := AsyncWorkerImpl{
		bcName: testBcName,
		filter: &protos.BlockFilter{
			BcName:   testBcName,
			Contract: `^\$`,
		},
		log:         tmpLog,
		finishTable: tmpFinishTable,
		close:       make(chan struct{}, 1),
	}

	return &aw, nil
}

func handleCreateChain(ctx base.TaskContext) error {
	return nil
}

func TestRegisterHandler(t *testing.T) {
	aw, err := newAsyncWorker()
	if err != nil {
		t.Fatalf("%+v\n", err)
	}
	aw.RegisterHandler(testContract, testEvent, handleCreateChain)
	if aw.methods[testContract] == nil {
		t.Fatalf("RegisterHandler register contract error")
	}
	if aw.methods[testContract][testEvent] == nil {
		t.Fatalf("RegisterHandler register event error")
	}
}

func TestGetAsyncTask(t *testing.T) {
	aw, err := newAsyncWorker()
	if err != nil {
		t.Fatalf("%+v\n", err)
	}
	_, err = aw.getAsyncTask("", testEvent)
	if err == nil {
		t.Fatalf("getAsyncTask error")
	}
	_, err = aw.getAsyncTask(testContract, testEvent)
	if err == nil {
		t.Fatalf("getAsyncTask error")
	}
	aw.RegisterHandler(testContract, testEvent, handleCreateChain)
	handler, err := aw.getAsyncTask(testContract, testEvent)
	if err != nil {
		t.Fatalf("getAsyncTask error")
	}
	ctx := newTaskContextImpl([]byte("hello"))
	if handler(ctx) != nil {
		t.Fatalf("getAsyncTask ctx error")
	}
}

func TestCursor(t *testing.T) {
	aw, err := newAsyncWorker()
	if err != nil {
		t.Fatalf("%+v\n", err)
	}

	_, err = aw.reloadCursor()
	if err != emptyErr {
		t.Fatalf("reload error, err=%v", err)
	}

	cursor := &asyncWorkerCursor{
		BlockHeight: 1,
		TxIndex:     int64(0),
		EventIndex:  int64(0),
	}
	cursorBuf, err := json.Marshal(cursor)
	if err != nil {
		t.Fatalf("marshal cursor failed when doAsyncTasks, err=%v", err)
	}
	aw.finishTable.Put([]byte(testBcName), cursorBuf)
	cursor, err = aw.reloadCursor()
	if err != nil {
		t.Fatalf("reloadCursor err=%v", err)
	}
	if cursor.BlockHeight != 1 || cursor.TxIndex != 0 || cursor.EventIndex != 0 {
		t.Fatalf("reloadCursor value error")
	}

	aw.storeCursor(asyncWorkerCursor{
		BlockHeight: 10,
	})
	cursor, err = aw.reloadCursor()
	if err != nil {
		t.Fatalf("reloadCursor err=%v", err)
	}
	if cursor.BlockHeight != 10 {
		t.Fatalf("reloadCursor value error")
	}
}

func TestDoAsyncTasks(t *testing.T) {
	aw, err := newAsyncWorker()
	if err != nil {
		t.Fatalf("%+v\n", err)
	}

	aw.RegisterHandler(testContract, testEvent, handleCreateChain)
	err = aw.doAsyncTasks(newTx(), 3, nil)
	if err != nil {
		t.Fatalf("doAsyncTasks error")
	}
	cursor, err := aw.reloadCursor()
	if err != nil {
		t.Fatalf("reloadCursor error")
	}
	if cursor.BlockHeight != 3 || cursor.TxIndex != 0 || cursor.EventIndex != 0 {
		t.Fatalf("doAsyncTasks block cursor error")
	}

	// 模拟中断存储cursor
	cursor = &asyncWorkerCursor{
		BlockHeight: 5,
		TxIndex:     int64(1),
		EventIndex:  int64(0),
	}
	cursorBuf, _ := json.Marshal(cursor)
	aw.finishTable.Put([]byte(testBcName), cursorBuf)
	aw.doAsyncTasks(newTxs(), 5, cursor)
	if cursor.BlockHeight != 5 || cursor.TxIndex != 1 || cursor.EventIndex != 0 {
		t.Fatalf("doAsyncTasks block break cursor error")
	}
}
