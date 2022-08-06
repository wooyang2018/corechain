package pow

import (
	"encoding/json"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/consensus/mock"
)

var (
	// 0x1903a30c
	target int64 = 419668748
	// 10进制545259519
	minTarget uint32 = 0x207FFFFF
)

func getPoWConsensusConf() []byte {
	j := `{
        	"defaultTarget": "419668748",
        	"adjustHeightGap": "2",
			"expectedPeriod":  "15",
			"maxTarget":       "0"
    	}`
	return []byte(j)
}

func prepare(config []byte) (*base.ConsensusCtx, error) {
	l := mock.NewFakeLedger(config)
	ps := powStorage{
		TargetBits: minTarget,
	}
	by, _ := json.Marshal(ps)
	l.SetConsensusStorage(1, by)
	l.SetConsensusStorage(2, by)
	ctx, err := mock.NewConsensusCtxWithCrypto(l)
	return ctx, err
}

func getConsensusConf(config []byte) base.ConsensusConfig {
	return base.ConsensusConfig{
		ConsensusName: "pow",
		Config:        string(config),
		StartHeight:   2,
		Index:         0,
	}
}

func TestNewPoWConsensus(t *testing.T) {
	ctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error")
	}
	conf := getConsensusConf(getPoWConsensusConf())
	i := NewPoWConsensus(*ctx, conf)
	if i == nil {
		t.Fatal("NewPoWConsensus error", "conf", conf)
	}
}

func TestProcessBeforeMiner(t *testing.T) {
	cctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error.")
	}
	i := NewPoWConsensus(*cctx, getConsensusConf(getPoWConsensusConf()))
	//更新下一次pow挖矿时的targetBits
	_, store, err := i.ProcessBeforeMiner(0, time.Now().UnixNano())
	if err != nil {
		t.Fatal("ProcessBeforeMiner error.")
	}
	t.Log(string(store))
}

func TestGetConsensusStatus(t *testing.T) {
	cctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error")
	}
	conf := getConsensusConf(getPoWConsensusConf())
	i := NewPoWConsensus(*cctx, conf)
	status, _ := i.GetConsensusStatus()
	if status.GetVersion() != 0 {
		t.Fatal("GetVersion error")
	}
	if status.GetStepConsensusIndex() != 0 {
		t.Fatal("GetStepConsensusIndex error")
	}
	if status.GetConsensusBeginInfo() != 2 {
		t.Fatal("GetConsensusBeginInfo error")
	}
	if status.GetConsensusName() != "pow" {
		t.Fatal("GetConsensusName error")
	}
	status.GetCurrentTerm()
	vb := status.GetCurrentValidatorsInfo()
	m := ValidatorsInfo{}
	err = json.Unmarshal(vb, &m)
	if err != nil {
		t.Fatal("GetCurrentValidatorsInfo unmarshal error", "error", err)
	}
	if m.Validators[0] != mock.Miner {
		t.Fatal("GetCurrentValidatorsInfo error", "address", m.Validators[0])
	}
}

func TestParseConsensusStorage(t *testing.T) {
	cctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error", err)
	}

	conf := getConsensusConf(getPoWConsensusConf())
	pow := NewPoWConsensus(*cctx, conf)

	ps := powStorage{
		TargetBits: uint32(target),
	}
	b, err := json.Marshal(ps)
	if err != nil {
		t.Fatal("ParseConsensusStorage Unmarshal error", "error", err)
	}

	b1, err := mock.NewBlockWithStorage(1, cctx.Crypto, cctx.Address, b)
	if err != nil {
		t.Fatal("NewBlockWithStorage error", err)
	}
	i, err := pow.ParseConsensusStorage(b1)
	if err != nil {
		t.Fatal("ParseConsensusStorage error", "error", err)
	}
	s, ok := i.(powStorage)
	if !ok {
		t.Fatal("ParseConsensusStorage transfer error")
	}
	if s.TargetBits != uint32(target) {
		t.Fatal("ParseConsensusStorage transfer error", "target", target)
	}
}

func TestSetCompact(t *testing.T) {
	bigint, pfNegative, pfOverflow := SetCompact(uint32(target))
	if pfNegative || pfOverflow {
		t.Fatal("TestSetCompact overflow or negative")
	}
	var strings []string
	for _, word := range bigint.Bits() {
		s := strconv.FormatUint(uint64(word), 16)
		strings = append(strings, s)
	}
	if bigint.BitLen() > 256 {
		t.Fatal("TestSetCompact overflow", "bigint.BitLen()", bigint.BitLen(), "string", strings)
	}
	// t := 0x0000000000000003A30C00000000000000000000000000000000000000000000, 对应target为0x1903a30c
	b := big.NewInt(0x0000000000000003A30C00000000)
	b.Lsh(b, 144)
	if b.Cmp(bigint) != 0 {
		t.Fatal("TestSetCompact equal err", "bigint", bigint, "b", b)
	}
}

func TestGetCompact(t *testing.T) {
	b := big.NewInt(0x0000000000000003A30C00000000)
	b.Lsh(b, 144)
	target, _ := GetCompact(b)
	if target != 0x1903a30c {
		t.Fatal("TestGetCompact error", "target", target)
	}
}

func TestIsProofed(t *testing.T) {
	cctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error", err)
	}
	conf := getConsensusConf(getPoWConsensusConf())
	i := NewPoWConsensus(*cctx, conf)
	pow, ok := i.(*PoWConsensus)
	if !ok {
		t.Fatal("TestIsProofed transfer error")
	}
	// t := 0x0000000000000003A30C00000000000000000000000000000000000000000000, 对应target为0x1903a30c
	b := big.NewInt(0x0000000000000003A30C00000000)
	b.Lsh(b, 144)
	blockid := b.Bytes()
	if !pow.IsProofed(blockid, pow.config.DefaultTarget) {
		t.Fatal("TestIsProofed error")
	}

	b = big.NewInt(1)
	b.Lsh(b, uint(4))
	blockid = b.Bytes()
	if !pow.IsProofed(blockid, pow.config.DefaultTarget) {
		t.Fatal("TestIsProofed error")
	}
}

func TestMining(t *testing.T) {
	cctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error", err)
	}
	conf := getConsensusConf(getPoWConsensusConf())
	i := NewPoWConsensus(*cctx, conf)
	pow, ok := i.(*PoWConsensus)
	if !ok {
		t.Fatal("TestMining transfer error")
	}
	pow.targetBits = minTarget
	pow.Start()
	defer pow.Stop()
	ps := powStorage{
		TargetBits: minTarget,
	}
	by, _ := json.Marshal(ps)
	block, err := mock.NewBlockWithStorage(3, cctx.Crypto, cctx.Address, by)
	if err != nil {
		t.Fatal("NewBlockWithStorage error", err)
	}
	err = pow.CalculateBlock(block)
	if err != nil {
		t.Fatal("CalculateBlock mining error", "err", err)
	}
	err = pow.ProcessConfirmBlock(block)
	if err != nil {
		t.Fatal("ProcessConfirmBlock mining error", "err", err)
	}
}

func TestRefreshDifficulty(t *testing.T) {
	cctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error", err)
	}
	conf := getConsensusConf(getPoWConsensusConf())
	i := NewPoWConsensus(*cctx, conf)
	pow, ok := i.(*PoWConsensus)
	if !ok {
		t.Fatal("TestRefreshDifficulty transfer error")
	}
	genesisBlock, err := mock.NewBlockWithStorage(0, cctx.Crypto, cctx.Address, []byte{})
	if err != nil {
		t.Fatal("NewBlock error", err)
	}
	l, ok := pow.Ledger.(*mock.FakeLedger)
	err = l.Put(genesisBlock)
	if err != nil {
		t.Fatal("TestRefreshDifficulty put genesis err", "err", err)
	}

	pow.targetBits = minTarget
	ps := powStorage{
		TargetBits: minTarget,
	}
	by, _ := json.Marshal(ps)
	block, err := mock.NewBlockWithStorage(3, cctx.Crypto, cctx.Address, by)
	if err != nil {
		t.Fatal("NewBlockWithStorage error", err)
	}
	task := mineTask{
		block: block,
		done:  make(chan error, 1),
		close: make(chan int, 1),
	}
	go pow.mining(&task)
	err = <-task.done
	if err != nil {
		t.Fatal("TestRefreshDifficulty mining error", "blockId", block.GetBlockid(), "err", err)
	}
	err = l.Put(block)
	if err != nil {
		t.Fatal("TestRefreshDifficulty put block err", "err", err)
	}
	block2, err := mock.NewBlockWithStorage(4, cctx.Crypto, cctx.Address, by)
	if err != nil {
		t.Fatal("NewBlockWithStorage error", err)
	}
	task2 := mineTask{
		block: block2,
		done:  make(chan error, 1),
		close: make(chan int, 1),
	}
	go pow.mining(&task2)
	err = <-task2.done
	if err != nil {
		t.Fatal("TestRefreshDifficulty mining error", "blockId", block2.GetBlockid(), "err", err)
	}
	err = l.Put(block2)
	if err != nil {
		t.Fatal("TestRefreshDifficulty put block err", "err", err)
	}

	target, err := pow.refreshDifficulty(block2.GetBlockid(), 5)
	if err != nil {
		t.Fatal("TestRefreshDifficulty refreshDifficulty err", "err", err, "target", target)
	}
	ps = powStorage{
		TargetBits: 218104063,
	}
	by, _ = json.Marshal(ps)
	block3, err := mock.NewBlockWithStorage(5, cctx.Crypto, cctx.Address, by)
	if err != nil {
		t.Fatal("NewBlockWithStorage error block3", err)
	}
	task3 := mineTask{
		block: block3,
		done:  make(chan error, 1),
		close: make(chan int, 1),
	}
	go pow.mining(&task3)
	task3.close <- 1
}

func TestCheckMinerMatch(t *testing.T) {
	cctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error", "error", err)
	}
	i := NewPoWConsensus(*cctx, getConsensusConf(getPoWConsensusConf()))
	if i == nil {
		t.Fatal("NewPoWConsensus error")
	}
	ps := powStorage{
		TargetBits: minTarget,
	}
	by, _ := json.Marshal(ps)
	block, err := mock.NewBlockWithStorage(3, cctx.Crypto, cctx.Address, by)
	c := cctx.BaseCtx
	check, err := i.CheckMinerMatch(&c, block)
	if err != nil {
		t.Fatal("CheckMinerMatch error", "err", err)
	}
	t.Log(check)
}

func TestCompeteMaster(t *testing.T) {
	cctx, err := prepare(getPoWConsensusConf())
	if err != nil {
		t.Fatal("prepare error", "error", err)
	}
	i := NewPoWConsensus(*cctx, getConsensusConf(getPoWConsensusConf()))
	master, sync, err := i.CompeteMaster(3)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(master, sync)
}
