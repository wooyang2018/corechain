package mimc

import (
	"github.com/consensys/gnark/backend/groth16"
	"github.com/wooyang2018/corechain/crypto/common/zkp"
)

// Setup generate CompiledConstraintSystem, ProvingKey and VerifyingKey
func Setup() (*zkp.ZkpInfo, error) {
	mimcCircuit, err := NewCircuit()
	if err != nil {
		return nil, err
	}

	pk, vk, err := groth16.Setup(mimcCircuit)
	if err != nil {
		return nil, err
	}

	zkpInfo := &zkp.ZkpInfo{
		R1CS:         mimcCircuit,
		ProvingKey:   pk,
		VerifyingKey: vk,
	}

	return zkpInfo, nil
}
