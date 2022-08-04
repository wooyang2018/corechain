package chainbft

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/wooyang2018/corechain/common/address"
	"github.com/wooyang2018/corechain/crypto/client/base"
	"github.com/wooyang2018/corechain/crypto/core/hash"
	"github.com/wooyang2018/corechain/protos"
)

type CBFTCrypto struct {
	Address      *address.Address
	CryptoClient base.CryptoClient
}

func NewCBFTCrypto(addr *address.Address, c base.CryptoClient) *CBFTCrypto {
	return &CBFTCrypto{
		Address:      addr,
		CryptoClient: c,
	}
}

func (c *CBFTCrypto) SignProposalMsg(msg *protos.ProposalMsg) (*protos.ProposalMsg, error) {
	msgDigest, err := MakeProposalMsgDigest(msg)
	if err != nil {
		return nil, err
	}
	msg.MsgDigest = msgDigest
	sign, err := c.CryptoClient.SignECDSA(c.Address.PrivateKey, msgDigest)
	if err != nil {
		return nil, err
	}
	msg.Sign = &protos.QuorumCertSign{
		Address:   c.Address.Address,
		PublicKey: c.Address.PublicKeyStr,
		Sign:      sign,
	}
	return msg, nil
}

func MakeProposalMsgDigest(msg *protos.ProposalMsg) ([]byte, error) {
	msgEncoder, err := encodeProposalMsg(msg)
	if err != nil {
		return nil, err
	}
	msg.MsgDigest = hash.DoubleSha256(msgEncoder)
	return hash.DoubleSha256(msgEncoder), nil
}

func encodeProposalMsg(msg *protos.ProposalMsg) ([]byte, error) {
	var msgBuf bytes.Buffer
	encoder := json.NewEncoder(&msgBuf)
	if err := encoder.Encode(msg.ProposalView); err != nil {
		return nil, err
	}
	if err := encoder.Encode(msg.ProposalId); err != nil {
		return nil, err
	}
	if err := encoder.Encode(msg.Timestamp); err != nil {
		return nil, err
	}
	if err := encoder.Encode(msg.JustifyQC); err != nil {
		return nil, err
	}
	return msgBuf.Bytes(), nil
}

func (c *CBFTCrypto) SignVoteMsg(msg []byte) (*protos.QuorumCertSign, error) {
	sign, err := c.CryptoClient.SignECDSA(c.Address.PrivateKey, msg)
	if err != nil {
		return nil, err
	}
	return &protos.QuorumCertSign{
		Address:   c.Address.Address,
		PublicKey: c.Address.PublicKeyStr,
		Sign:      sign,
	}, nil
}

func (c *CBFTCrypto) VerifyVoteMsgSign(sig *protos.QuorumCertSign, msg []byte) (bool, error) {
	ak, err := c.CryptoClient.GetEcdsaPublicKeyFromJsonStr(sig.GetPublicKey())
	if err != nil {
		return false, err
	}
	addr, err := c.CryptoClient.GetAddressFromPublicKey(ak)
	if err != nil {
		return false, err
	}
	if addr != sig.GetAddress() {
		return false, errors.New("VerifyVoteMsgSign error, addr not match pk: " + addr)
	}
	return c.CryptoClient.VerifyECDSA(ak, sig.GetSign(), msg)
}
