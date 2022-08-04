package mimc

import (
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
)

// ZkpInfo includes CompiledConstraintSystem、ProvingKey、VerifyingKey
type ZkpInfo struct {
	R1CS         frontend.CompiledConstraintSystem
	ProvingKey   groth16.ProvingKey
	VerifyingKey groth16.VerifyingKey
}

// Setup generate CompiledConstraintSystem, ProvingKey and VerifyingKey
func Setup() (*ZkpInfo, error) {
	mimcCircuit, err := NewCircuit()
	if err != nil {
		return nil, err
	}

	pk, vk, err := groth16.Setup(mimcCircuit)
	if err != nil {
		return nil, err
	}

	zkpInfo := &ZkpInfo{
		R1CS:         mimcCircuit,
		ProvingKey:   pk,
		VerifyingKey: vk,
	}

	return zkpInfo, nil
}
