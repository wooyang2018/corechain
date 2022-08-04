package mimc

import (
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/examples/mimc"
	"github.com/consensys/gnark/frontend"
	"github.com/wooyang2018/corechain/crypto/core/hash"
)

// Prove generate a zkp proof using ProvingKey
func Prove(ccs frontend.CompiledConstraintSystem, pk groth16.ProvingKey, secret []byte) (groth16.Proof, error) {
	hashResult := hash.HashUsingDefaultMiMC(secret)
	assignment := &mimc.Circuit{
		PreImage: frontend.Value(secret),
		Hash:     frontend.Value(hashResult),
	}

	proof, err := groth16.Prove(ccs, pk, assignment)
	if err != nil {
		return nil, err
	}

	return proof, nil
}
