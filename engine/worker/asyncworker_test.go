package worker

import (
	"encoding/json"
	"github.com/wooyang2018/corechain/storage/leveldb"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/ledger"
	lconf "github.com/wooyang2018/corechain/ledger/config"
	"github.com/wooyang2018/corechain/logger"
	mock "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/protos"
	"github.com/wooyang2018/corechain/storage"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

const (
	testBcName = "parachain"
)

func newTx() []*protos.FilteredTransaction {
	var txs []*protos.FilteredTransaction
	txs = append(txs, &protos.FilteredTransaction{
		Txid: "txid_1",
		Events: []*protos.ContractEvent{
			{
				Contract: "$parachain",
				Name:     "CreateBlockChain",
				Body:     []byte("hello2"),
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
				Contract: "$parachain",
				Name:     "CreateBlockChain",
				Body:     []byte("hello3"),
			},
		},
	})
	return txs
}

type TestHelper struct {
	basedir string
	db      storage.Database
	log     logger.Logger
}

func NewTestHelper() (*TestHelper, error) {
	basedir, err := ioutil.TempDir("", "asyncworker")
	lcfg, err := lconf.LoadLedgerConf(mock.GetLedgerConfFilePath())
	if err != nil {
		return nil, err
	}
	// 目前仅使用默认设置
	kvParam := &leveldb.KVParameter{
		DBPath:                filepath.Join(basedir, "database"),
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
	econf, err := mock.GetMockEnvConf()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(basedir, "log")
	logger.InitMLog(econf.GenConfFilePath(econf.LogConf), logPath)
	log, _ := logger.NewLogger("", "asyncworker")

	th := &TestHelper{
		basedir: basedir,
		db:      tmpFinishTable,
		log:     log,
	}
	return th, nil
}

func (th *TestHelper) close() {
	th.db.Close()
	os.RemoveAll(th.basedir)
}

//新建异步服务代理
func newAsyncWorker() *AsyncWorkerImpl {
	aw := AsyncWorkerImpl{
		bcname: testBcName,
		filter: &protos.BlockFilter{
			Bcname:   testBcName,
			Contract: `^\$`,
		},
		close: make(chan struct{}, 1),
	}
	return &aw
}

func handleCreateChain(ctx base.TaskContext) error {
	return nil
}

func TestRegisterHandler(t *testing.T) {
	aw := newAsyncWorker()
	th, err := NewTestHelper()
	if err != nil {
		t.Errorf("NewTestHelper error")
	}
	defer th.close()
	aw.finishTable = th.db
	aw.log = th.log
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	if aw.methods["$parachain"] == nil {
		t.Errorf("RegisterHandler register contract error")
		return
	}
	if aw.methods["$parachain"]["CreateBlockChain"] == nil {
		t.Errorf("RegisterHandler register event error")
	}
	aw.RegisterHandler("", "", handleCreateChain)
	aw.RegisterHandler("parachain", "CreateBlockChain", handleCreateChain)
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
}

func TestGetAsyncTask(t *testing.T) {
	aw := newAsyncWorker()
	_, err := aw.getAsyncTask("", "CreateBlockChain")
	if err == nil {
		t.Errorf("getAsyncTask error")
		return
	}
	_, err = aw.getAsyncTask("$parachain", "CreateBlockChain")
	if err == nil {
		t.Errorf("getAsyncTask error")
		return
	}
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	handler, err := aw.getAsyncTask("$parachain", "CreateBlockChain")
	if err != nil {
		t.Errorf("getAsyncTask error")
		return
	}
	ctx := newTaskContextImpl([]byte("hello"))
	if handler(ctx) != nil {
		t.Errorf("getAsyncTask ctx error")
		return
	}
}

func TestCursor(t *testing.T) {
	aw := newAsyncWorker()
	th, err := NewTestHelper()
	if err != nil {
		t.Errorf("NewTestHelper error")
	}
	defer th.close()
	aw.finishTable = th.db
	aw.log = th.log
	_, err = aw.reloadCursor()
	if err != emptyErr {
		t.Errorf("reload error, err=%v", err)
		return
	}
	// 执行完毕后进行持久化
	cursor := &asyncWorkerCursor{
		BlockHeight: 1,
		TxIndex:     int64(0),
		EventIndex:  int64(0),
	}
	cursorBuf, err := json.Marshal(cursor)
	if err != nil {
		t.Errorf("marshal cursor failed when doAsyncTasks, err=%v", err)
		return
	}
	aw.finishTable.Put([]byte(testBcName), cursorBuf)
	cursor, err = aw.reloadCursor()
	if err != nil {
		t.Errorf("reloadCursor err=%v", err)
		return
	}
	if cursor.BlockHeight != 1 || cursor.TxIndex != 0 || cursor.EventIndex != 0 {
		t.Errorf("reloadCursor value error")
		return
	}
	aw.storeCursor(asyncWorkerCursor{
		BlockHeight: 10,
	})
}

func TestDoAsyncTasks(t *testing.T) {
	aw := newAsyncWorker()
	th, err := NewTestHelper()
	if err != nil {
		t.Errorf("NewTestHelper error")
	}
	defer th.close()
	aw.finishTable = th.db
	aw.log = th.log
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	err = aw.doAsyncTasks(newTx(), 3, nil)
	if err != nil {
		t.Errorf("doAsyncTasks error")
		return
	}
	cursor, err := aw.reloadCursor()
	if err != nil {
		t.Errorf("reloadCursor error")
		return
	}
	if cursor.BlockHeight != 3 || cursor.TxIndex != 0 || cursor.EventIndex != 0 {
		t.Errorf("doAsyncTasks block cursor error")
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
		t.Errorf("doAsyncTasks block break cursor error")
	}
}

func TestStartAsyncTask(t *testing.T) {
	aw := newAsyncWorker()
	th, err := NewTestHelper()
	if err != nil {
		t.Errorf("NewTestHelper error")
	}
	defer th.close()
	aw.finishTable = th.db
	aw.log = th.log
	aw.RegisterHandler("$parachain", "CreateBlockChain", handleCreateChain)
	aw.Start()
	aw.Stop()
}
