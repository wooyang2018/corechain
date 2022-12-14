package bridge

import (
	"github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/protos"
)

type VMConfig interface {
	DriverName() string
	IsEnable() bool
}

// ContractCodeProvider provides source code and desc of contract
type ContractCodeProvider interface {
	GetContractCodeDesc(name string) (*protos.WasmCodeDesc, error)
	GetContractCode(name string) ([]byte, error)
	GetContractAbi(name string) ([]byte, error)
	GetContractCodeFromCache(name string) ([]byte, error)
	GetContractAbiFromCache(name string) ([]byte, error)
}

// InstanceCreator is the creator of contract virtual machine instance
type InstanceCreator interface {
	// CreateInstance instances a wasm virtual machine instance which can run a single contract call
	CreateInstance(ctx *Context, cp ContractCodeProvider) (Instance, error)
	RemoveCache(name string)
}

// Instance is a contract virtual machine instance which can run a single contract call
type Instance interface {
	Exec() error
	ResourceUsed() base.Limits
	Release()
	Abort(msg string)
}
