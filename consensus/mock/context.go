package mock

import (
	"errors"
	"math/big"

	"github.com/wooyang2018/corechain/common/address"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/utils"
	cbase "github.com/wooyang2018/corechain/consensus/base"
	"github.com/wooyang2018/corechain/contract"
	"github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/crypto/client/base"
	"github.com/wooyang2018/corechain/logger"
	mockConf "github.com/wooyang2018/corechain/mock/config"
	"github.com/wooyang2018/corechain/protos"
)

var (
	BcName = "corechain"
	nodeIp = "/ip4/127.0.0.1/tcp/47101/p2p/QmVcSF4F7rTdsvUJqsik98tXRXMBUqL5DSuBpyYKVhjuG4"
	priKey = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571,"D":29079635126530934056640915735344231956621504557963207107451663058887647996601}`
	PubKey = `{"Curvname":"P-256","X":74695617477160058757747208220371236837474210247114418775262229497812962582435,"Y":51348715319124770392993866417088542497927816017012182211244120852620959209571}`
	Miner  = "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"

	blockSetItemErr = errors.New("item invalid")
)

type FakeKContext struct {
	args map[string][]byte
	m    map[string]map[string][]byte
}

func NewFakeKContext(args map[string][]byte, m map[string]map[string][]byte) *FakeKContext {
	return &FakeKContext{
		args: args,
		m:    m,
	}
}

func (c *FakeKContext) EmitAsyncTask(event string, args interface{}) error {
	return nil
}

func (c *FakeKContext) Args() map[string][]byte {
	return c.args
}

func (c *FakeKContext) Initiator() string {
	return "TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY"
}

func (c *FakeKContext) Caller() string {
	return ""
}

func (c *FakeKContext) AuthRequire() []string {
	return []string{"TeyyPLpp9L7QAcxHangtcHTu7HUZ6iydY", "SmJG3rH2ZzYQ9ojxhbRCPwFiE9y6pD1Co"}
}

func (c *FakeKContext) GetAccountAddresses(accountName string) ([]string, error) {
	return nil, nil
}

func (c *FakeKContext) VerifyContractPermission(initiator string, authRequire []string, contractName string, methodName string) (bool, error) {
	return true, nil
}

func (c *FakeKContext) VerifyContractOwnerPermission(contractName string, authRequire []string) error {
	return nil
}

func (c *FakeKContext) RWSet() *contract.RWSet {
	return nil
}

func (c *FakeKContext) AddEvent(events ...*protos.ContractEvent) {}

func (c *FakeKContext) Flush() error {
	return nil
}

func (c *FakeKContext) Get(bucket string, key []byte) ([]byte, error) {
	if _, ok := c.m[bucket]; !ok {
		return nil, nil
	}
	return c.m[bucket][utils.F(key)], nil
}

func (c *FakeKContext) Select(bucket string, startKey []byte, endKey []byte) (contract.Iterator, error) {
	return nil, nil
}

func (c *FakeKContext) Put(bucket string, key, value []byte) error {
	if _, ok := c.m[bucket]; !ok {
		a := make(map[string][]byte)
		a[utils.F(key)] = value
		c.m[bucket] = a
	}
	c.m[bucket][utils.F(key)] = value
	return nil
}

func (c *FakeKContext) Del(bucket string, key []byte) error {
	return nil
}

func (c *FakeKContext) AddResourceUsed(delta contract.Limits) {}

func (c *FakeKContext) ResourceLimit() contract.Limits {
	return contract.Limits{
		Cpu:    0,
		Memory: 0,
		Disk:   0,
		XFee:   0,
	}
}

func (c *FakeKContext) Call(module, contract, method string, args map[string][]byte) (*contract.Response, error) {
	return nil, nil
}

func (c *FakeKContext) UTXORWSet() *contract.UTXORWSet {
	return &contract.UTXORWSet{
		Rset: []*protos.TxInput{},
		WSet: []*protos.TxOutput{},
	}
}
func (c *FakeKContext) Transfer(from string, to string, amount *big.Int) error {
	return nil
}
func (c *FakeKContext) QueryBlock(blockid []byte) (*protos.InternalBlock, error) {
	return &protos.InternalBlock{}, nil
}
func (c *FakeKContext) QueryTransaction(txid []byte) (*protos.Transaction, error) {
	return &protos.Transaction{}, nil
}

//FakeManager 合约Manager
type FakeManager struct {
	R *FakeRegistry
}

func (m *FakeManager) NewContext(cfg *contract.ContextConfig) (contract.Context, error) {
	return nil, nil
}

func (m *FakeManager) NewStateSandbox(cfg *contract.SandboxConfig) (contract.StateSandbox, error) {
	return nil, nil
}

func (m *FakeManager) GetKernRegistry() contract.KernRegistry {
	return m.R
}

type FakeRegistry struct {
	M map[string]contract.KernMethod
}

func (r *FakeRegistry) RegisterKernMethod(contract, method string, handler contract.KernMethod) {
	r.M[method] = handler
}

func (r *FakeRegistry) UnregisterKernMethod(ctract, method string) {
	return
}

func (r *FakeRegistry) GetKernMethod(contract, method string) (contract.KernMethod, error) {
	return nil, nil
}

func (r *FakeRegistry) RegisterShortcut(oldmethod, contract, method string) {
}

func NewCryptoClient() (base.CryptoClient, *address.Address, error) {
	cc, err := client.CreateCryptoClientFromJSONPrivateKey([]byte(priKey))
	if err != nil {
		return nil, nil, err
	}
	sk, err := cc.GetEcdsaPrivateKeyFromJsonStr(priKey)
	if err != nil {
		return nil, nil, err
	}
	pk, err := cc.GetEcdsaPublicKeyFromJsonStr(PubKey)
	if err != nil {
		return nil, nil, err
	}
	a := &address.Address{
		Address:       Miner,
		PrivateKeyStr: priKey,
		PublicKeyStr:  PubKey,
		PrivateKey:    sk,
		PublicKey:     pk,
	}
	return cc, a, nil
}

// NewConsensusCtxWithCrypto 返回除ledger以外的所有所需的共识上下文
func NewConsensusCtxWithCrypto(ledger *FakeLedger) (*cbase.ConsensusCtx, error) {
	cc, a, err := NewCryptoClient()
	if err != nil {
		return nil, err
	}
	ctx := NewConsensusCtx(ledger)
	ctx.Crypto = cc
	ctx.Address = a
	return &ctx, nil
}

func NewConsensusCtx(ledger *FakeLedger) cbase.ConsensusCtx {
	mockConf.InitFakeLogger()
	log, _ := logger.NewLogger("", "consensus_test")
	ctx := cbase.ConsensusCtx{
		BcName: "corechain",
		Ledger: ledger,
		BaseCtx: xctx.BaseCtx{
			XLog: log,
		},
		Contract: &FakeManager{
			R: &FakeRegistry{
				M: make(map[string]contract.KernMethod),
			},
		},
	}
	return ctx
}
