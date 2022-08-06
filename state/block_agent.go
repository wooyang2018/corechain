package state

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/wooyang2018/corechain/ledger"
	"github.com/wooyang2018/corechain/protos"
)

var _ ledger.BlockHandle = (*BlockAgent)(nil)

type BlockAgent struct {
	blk *protos.InternalBlock
}

//ConsensusStorage 共识部分字段分开存储在区块中
type ConsensusStorage struct {
	TargetBits  int32              `json:"targetBits,omitempty"`
	Justify     *protos.QuorumCert `json:"justify,omitempty"`
	CurTerm     int64              `json:"curTerm,omitempty"`
	CurBlockNum int64              `json:"curBlockNum,omitempty"`
}

func NewBlockAgent(blk *protos.InternalBlock) *BlockAgent {
	return &BlockAgent{
		blk: blk,
	}
}

func (t *BlockAgent) GetProposer() []byte {
	return t.blk.GetProposer()
}

func (t *BlockAgent) GetHeight() int64 {
	return t.blk.GetHeight()
}

func (t *BlockAgent) GetBlockid() []byte {
	return t.blk.GetBlockid()
}

//GetConsensusStorage 获取共识记录信息
func (t *BlockAgent) GetConsensusStorage() ([]byte, error) {
	strg := &ConsensusStorage{
		TargetBits: t.blk.GetTargetBits(),
		Justify:    t.blk.GetJustify(),
		CurTerm:    t.blk.GetCurTerm(),
	}

	js, err := json.Marshal(strg)
	if err != nil {
		return nil, fmt.Errorf("json marshal failed.err:%v", err)
	}

	return js, nil
}

//GetTimestamp 获取区块的时间戳
func (t *BlockAgent) GetTimestamp() int64 {
	return t.blk.GetTimestamp()
}

//SetItem 用于pow挖矿时更新nonce，或者设置blockid和sign
func (t *BlockAgent) SetItem(item string, value interface{}) error {
	switch item {
	case "nonce":
		nonce, ok := value.(int32)
		if !ok {
			return fmt.Errorf("nonce type not match")
		}
		t.blk.Nonce = nonce
	case "blockid":
		blockId, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("blockid type not match")
		}
		t.blk.Blockid = blockId
	case "sign":
		sign, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("sign type not match")
		}
		t.blk.Sign = sign
	default:
		return fmt.Errorf("item not support set")
	}

	return nil
}

//MakeBlockId 设置并返回BlockId
func (t *BlockAgent) MakeBlockId() ([]byte, error) {
	blkId, err := ledger.MakeBlockID(t.blk)
	if err != nil {
		return nil, err
	}
	t.blk.Blockid = blkId
	return blkId, nil
}

func (t *BlockAgent) GetPreHash() []byte {
	return t.blk.GetPreHash()
}

func (t *BlockAgent) GetPublicKey() string {
	return string(t.blk.GetPubkey())
}

func (t *BlockAgent) GetSign() []byte {
	return t.blk.GetSign()
}

func (t *BlockAgent) GetInTrunk() bool {
	return t.blk.InTrunk
}

func (t *BlockAgent) GetNextHash() []byte {
	return t.blk.NextHash
}

//GetTxIDs 获取区块中所有交易的id列表
func (t *BlockAgent) GetTxIDs() []string {
	txIDs := []string{}
	for _, tx := range t.blk.Transactions {
		txIDs = append(txIDs, hex.EncodeToString(tx.Txid))
	}
	return txIDs

}
