package mock

import (
	"time"

	"github.com/wooyang2018/corechain/common/address"
	"github.com/wooyang2018/corechain/crypto/client/base"
)

type FakeBlock struct {
	Proposer         string
	Height           int64
	Blockid          []byte
	ConsensusStorage []byte
	Timestamp        int64
	Nonce            int32
	PublicKey        string
	Sign             []byte
	PreHash          []byte
}

func NewBlockWithStorage(height int, c base.CryptoClient, a *address.Address, s []byte) (*FakeBlock, error) {
	b := &FakeBlock{
		Proposer:         a.Address,
		Height:           int64(height),
		Blockid:          []byte{byte(height)},
		ConsensusStorage: s,
		Timestamp:        time.Now().UnixNano(),
		PublicKey:        a.PrivateKeyStr,
		PreHash:          []byte{byte(height - 1)},
	}
	s, err := c.SignECDSA(a.PrivateKey, b.Blockid)
	if err == nil {
		b.Sign = s
	}
	return b, err
}

func NewFakeBlock(height int) *FakeBlock {
	return &FakeBlock{
		Height:           int64(height),
		Blockid:          []byte{byte(height)},
		ConsensusStorage: []byte{},
		Timestamp:        time.Now().UnixNano(),
		Proposer:         "dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN",
		PreHash:          []byte{byte(height - 1)},
	}
}

func (b *FakeBlock) MakeBlockId() ([]byte, error) {
	return b.Blockid, nil
}

func (b *FakeBlock) SetTimestamp(t int64) {
	b.Timestamp = t
}

func (b *FakeBlock) SetProposer(m string) {
	b.Proposer = m
}

func (b *FakeBlock) SetItem(param string, value interface{}) error {
	switch param {
	case "nonce":
		if s, ok := value.(int32); ok {
			b.Nonce = s
			return nil
		}
	}
	return blockSetItemErr
}

func (b *FakeBlock) GetProposer() []byte {
	return []byte(b.Proposer)
}

func (b *FakeBlock) GetHeight() int64 {
	return b.Height
}

func (b *FakeBlock) GetPreHash() []byte {
	return b.PreHash
}

func (b *FakeBlock) GetBlockid() []byte {
	return b.Blockid
}

func (b *FakeBlock) GetPublicKey() string {
	return b.PublicKey
}
func (b *FakeBlock) GetSign() []byte {
	return b.Sign
}

func (b *FakeBlock) GetConsensusStorage() ([]byte, error) {
	return b.ConsensusStorage, nil
}

func (b *FakeBlock) GetTimestamp() int64 {
	return b.Timestamp
}

func (b *FakeBlock) GetInTrunk() bool {
	return false
}
func (b *FakeBlock) GetNextHash() []byte {
	return []byte{}
}
func (b *FakeBlock) GetTxIDs() []string {
	return []string{}
}
