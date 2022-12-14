package signature

import (
	"crypto/ecdsa"
	"encoding/asn1"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wooyang2018/corechain/crypto/common/types"
	"github.com/wooyang2018/corechain/crypto/core/multisign"
	"github.com/wooyang2018/corechain/crypto/core/schnorr_ring_sign"
	"github.com/wooyang2018/corechain/crypto/core/schnorr_sign"
	"github.com/wooyang2018/corechain/crypto/core/sign"
	"github.com/wooyang2018/corechain/crypto/core/threshold/schnorr/tss_sign"
)

var (
	InvalidInputParamsError        = errors.New("Invalid input params")
	NotExactTheSameCurveInputError = errors.New("The private keys of all the keys are not using the the same curve")
	TooSmallNumOfkeysError         = errors.New("The total num of keys should be greater than one")
	EmptyMessageError              = errors.New("Message to be sign should not be nil")
	InValidSignatureError          = errors.New("XuperSignature is invalid")
)

func XuperSigVerify(keys []*ecdsa.PublicKey, signature, message []byte) (bool, error) {
	//	xuperSig, err := unmarshalXuperSignature(signature)
	xuperSig := new(types.XuperSignature)
	err := json.Unmarshal(signature, xuperSig)

	// 说明不是统一超级签名的格式
	if err != nil {
		switch keys[0].Params().Name {
		case types.CurveNist: // NIST
			verifyResult, err := sign.VerifyECDSA(keys[0], signature, message)
			return verifyResult, err
		case types.CurveGm: // 国密
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		default: // 不支持的密码学类型
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		}

		return false, err
	}

	switch xuperSig.SigType {
	// ECDSA签名
	case types.ECDSA:
		switch keys[0].Params().Name {
		case types.CurveNist: // NIST
			verifyResult, err := sign.VerifyV2ECDSA(keys[0], xuperSig.SigContent, message)
			return verifyResult, err
		case types.CurveGm: // 国密
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		default: // 不支持的密码学类型
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		}
	// Schnorr签名
	case types.Schnorr:
		switch keys[0].Params().Name {
		case types.CurveNist: // NIST
			verifyResult, err := schnorr_sign.Verify(keys[0], xuperSig.SigContent, message)
			return verifyResult, err
		case types.CurveGm: // 国密
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		default: // 不支持的密码学类型
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		}
	// Schnorr环签名
	case types.SchnorrRing:
		switch keys[0].Params().Name {
		case types.CurveNist: // NIST
			verifyResult, err := schnorr_ring_sign.Verify(keys, xuperSig.SigContent, message)
			return verifyResult, err
		case types.CurveGm: // 国密
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		default: // 不支持的密码学类型
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		}
	// 多重签名
	case types.MultiSig:
		switch keys[0].Params().Name {
		case types.CurveNist: // NIST
			verifyResult, err := multisign.VerifyMultiSig(keys, xuperSig.SigContent, message)
			return verifyResult, err
		case types.CurveGm: // 国密
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		default: // 不支持的密码学类型
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		}
	case types.TssSig:
		switch keys[0].Params().Name {
		case types.CurveNist: // NIST
			verifyResult, err := tss_sign.VerifyTssSig(keys[0], xuperSig.SigContent, message)
			return verifyResult, err
		case types.CurveGm: // 国密
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		default: // 不支持的密码学类型
			return false, fmt.Errorf("This cryptography[%v] has not been supported yet.", keys[0].Params().Name)
		}
	// 不支持的签名类型
	default:
		err = fmt.Errorf("This XuperSignature type[%v] is not supported in this version.", xuperSig.SigType)
		return false, err
	}

	return false, nil
}

func unmarshalXuperSignature(rawSig []byte) (*types.XuperSignature, error) {
	sig := new(types.XuperSignature)
	_, err := asn1.Unmarshal(rawSig, sig)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmashal xuper signature [%s]", err)
	}

	// validate xuper sig format
	if sig.SigContent == nil {
		return nil, InValidSignatureError
	}

	return sig, nil
}
