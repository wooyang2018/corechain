package base

import (
	"github.com/wooyang2018/corechain/protos"
)

type AclManager interface {
	GetAccountACL(accountName string) (*protos.Acl, error)
	GetContractMethodACL(contractName, methodName string) (*protos.Acl, error)
	GetAccountAddresses(accountName string) ([]string, error)
}
