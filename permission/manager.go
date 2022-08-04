package permission

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wooyang2018/corechain/permission/base"
	pctx "github.com/wooyang2018/corechain/permission/context"
	"github.com/wooyang2018/corechain/protos"
)

// Manager manages all ACL releated data, providing read/write interface for ACL table
type Manager struct {
	Ctx *pctx.AclCtx
}

// NewACLManager create instance of ACLManager
func NewACLManager(ctx *pctx.AclCtx) (base.AclManager, error) {
	if ctx == nil || ctx.Ledger == nil || ctx.Contract == nil || ctx.BcName == "" {
		return nil, fmt.Errorf("acl ctx set error")
	}

	newAccountGas, err := ctx.Ledger.GetNewAccountGas()
	if err != nil {
		return nil, fmt.Errorf("get create account gas failed.err:%v", err)
	}

	t := NewKernContractMethod(ctx.BcName, newAccountGas)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(base.SubModName, "NewAccount", t.NewAccount)
	register.RegisterKernMethod(base.SubModName, "SetAccountAcl", t.SetAccountACL)
	register.RegisterKernMethod(base.SubModName, "SetMethodAcl", t.SetMethodACL)
	register.RegisterShortcut("NewAccount", base.SubModName, "NewAccount")
	register.RegisterShortcut("SetAccountAcl", base.SubModName, "SetAccountAcl")
	register.RegisterShortcut("SetMethodAcl", base.SubModName, "SetMethodAcl")

	mg := &Manager{
		Ctx: ctx,
	}

	return mg, nil
}

// GetAccountACL get acl of an account
func (mgr *Manager) GetAccountACL(accountName string) (*protos.Acl, error) {
	acl, err := mgr.GetObjectBySnapshot(base.GetAccountBucket(), []byte(accountName))
	if err != nil {
		return nil, fmt.Errorf("query account acl failed.err:%v", err)
	}

	if len(acl) <= 0 {
		return nil, nil
	}

	aclBuf := &protos.Acl{}
	err = json.Unmarshal(acl, aclBuf)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal acl failed.acl:%s,err:%v", string(acl), err)
	}
	return aclBuf, nil
}

// GetContractMethodACL get acl of contract method
func (mgr *Manager) GetContractMethodACL(contractName, methodName string) (*protos.Acl, error) {
	key := base.MakeContractMethodKey(contractName, methodName)
	acl, err := mgr.GetObjectBySnapshot(base.GetContractBucket(), []byte(key))
	if err != nil {
		return nil, fmt.Errorf("query contract method acl failed.err:%v", err)
	}

	if len(acl) <= 0 {
		return nil, nil
	}

	aclBuf := &protos.Acl{}
	err = json.Unmarshal(acl, aclBuf)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal acl failed.acl:%s,err:%v", string(acl), err)
	}
	return aclBuf, nil
}

// GetAccountAddresses get the addresses belongs to contract account
func (mgr *Manager) GetAccountAddresses(accountName string) ([]string, error) {
	acl, err := mgr.GetAccountACL(accountName)
	if err != nil {
		return nil, err
	}

	return mgr.getAddressesByACL(acl)
}

func (mgr *Manager) GetObjectBySnapshot(bucket string, object []byte) ([]byte, error) {
	// 根据tip blockid 创建快照
	reader, err := mgr.Ctx.Ledger.GetTipXMSnapshotReader()
	if err != nil {
		return nil, err
	}

	return reader.Get(bucket, object)
}

func (mgr *Manager) getAddressesByACL(acl *protos.Acl) ([]string, error) {
	addresses := make([]string, 0)

	switch acl.GetPm().GetRule() {
	case protos.PermissionRule_SIGN_THRESHOLD:
		for ak := range acl.GetAksWeight() {
			addresses = append(addresses, ak)
		}
	case protos.PermissionRule_SIGN_AKSET:
		for _, set := range acl.GetAkSets().GetSets() {
			aks := set.GetAks()
			addresses = append(addresses, aks...)
		}
	default:
		return nil, errors.New("Unknown permission rule")
	}

	return addresses, nil
}
