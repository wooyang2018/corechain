package zkp

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
