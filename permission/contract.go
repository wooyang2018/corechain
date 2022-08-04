package permission

import (
	"encoding/json"
	"fmt"

	"github.com/wooyang2018/corechain/contract"
	"github.com/wooyang2018/corechain/contract/sandbox"
	"github.com/wooyang2018/corechain/permission/base"
	"github.com/wooyang2018/corechain/protos"
)

type KernMethod struct {
	BcName                   string
	NewAccountResourceAmount int64
}

func NewKernContractMethod(bcName string, NewAccountResourceAmount int64) *KernMethod {
	t := &KernMethod{
		BcName:                   bcName,
		NewAccountResourceAmount: NewAccountResourceAmount,
	}
	return t
}

func (t *KernMethod) NewAccount(ctx contract.KContext) (*contract.Response, error) {
	if ctx.ResourceLimit().XFee < t.NewAccountResourceAmount {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewAccountResourceAmount)
	}
	args := ctx.Args()
	// json -> protos.Acl
	accountName := args["account_name"]
	aclJSON := args["acl"]
	aclBuf := &protos.Acl{}
	err := json.Unmarshal(aclJSON, aclBuf)
	if err != nil {
		return nil, fmt.Errorf("unmarshal args acl error: %v", err)
	}

	if accountName == nil {
		return nil, fmt.Errorf("Invoke NewAccount failed, warn: account name is empty")
	}
	accountStr := string(accountName)
	if validErr := base.ValidRawAccount(accountStr); validErr != nil {
		return nil, validErr
	}

	bcname := t.BcName
	if bcname == "" {
		return nil, fmt.Errorf("block name is empty")
	}
	accountStr = base.MakeAccountKey(bcname, accountStr)

	if validErr := validACL(aclBuf); validErr != nil {
		return nil, validErr
	}

	oldAccount, err := ctx.Get(base.GetAccountBucket(), []byte(accountStr))
	if err != nil && err != sandbox.ErrNotFound {
		return nil, err
	}
	if oldAccount != nil {
		return nil, fmt.Errorf("account already exists: %s", accountName)
	}
	err = ctx.Put(base.GetAccountBucket(), []byte(accountStr), aclJSON)
	if err != nil {
		return nil, err
	}

	// add ak -> account reflection
	err = UpdateAK2AccountReflection(ctx, nil, aclJSON, accountStr)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: t.NewAccountResourceAmount,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  base.StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}

func (t *KernMethod) SetAccountACL(ctx contract.KContext) (*contract.Response, error) {
	if ctx.ResourceLimit().XFee < t.NewAccountResourceAmount/1000 {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewAccountResourceAmount/1000)
	}
	args := ctx.Args()
	// json -> protos.Acl
	accountName := args["account_name"]
	aclJSON := args["acl"]
	aclBuf := &protos.Acl{}
	err := json.Unmarshal(aclJSON, aclBuf)
	if err != nil {
		return nil, fmt.Errorf("unmarshal args acl error: %v", err)
	}

	if validErr := validACL(aclBuf); validErr != nil {
		return nil, validErr
	}

	data, err := ctx.Get(base.GetAccountBucket(), accountName)
	if err != nil {
		return nil, err
	}
	// delete ak -> account reflection
	// add ak -> account reflection
	aclOldJSON := data
	err = UpdateAK2AccountReflection(ctx, aclOldJSON, aclJSON, string(accountName))
	if err != nil {
		return nil, err
	}

	err = ctx.Put(base.GetAccountBucket(), accountName, aclJSON)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: t.NewAccountResourceAmount / 1000,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  base.StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}

func (t *KernMethod) SetMethodACL(ctx contract.KContext) (*contract.Response, error) {
	if ctx.ResourceLimit().XFee < t.NewAccountResourceAmount/1000 {
		return nil, fmt.Errorf("gas not enough, expect no less than %d", t.NewAccountResourceAmount/1000)
	}
	args := ctx.Args()
	contractNameBuf := args["contract_name"]
	methodNameBuf := args["method_name"]
	if contractNameBuf == nil || methodNameBuf == nil {
		return nil, fmt.Errorf("set method acl failed, contract name is nil or method name is nil")
	}

	// json -> protos.Acl
	contractName := string(contractNameBuf)
	methodName := string(methodNameBuf)
	aclJSON := args["acl"]
	aclBuf := &protos.Acl{}
	err := json.Unmarshal(aclJSON, aclBuf)
	if err != nil {
		return nil, fmt.Errorf("unmarshal args acl error: %v", err)
	}

	if validErr := validACL(aclBuf); validErr != nil {
		return nil, validErr
	}
	key := base.MakeContractMethodKey(contractName, methodName)
	err = ctx.Put(base.GetContractBucket(), []byte(key), aclJSON)
	if err != nil {
		return nil, err
	}

	delta := contract.Limits{
		XFee: t.NewAccountResourceAmount / 1000,
	}
	ctx.AddResourceUsed(delta)

	return &contract.Response{
		Status:  base.StatusOK,
		Message: "success",
		Body:    aclJSON,
	}, nil
}

func validACL(acl *protos.Acl) error {
	// param absence check
	if acl == nil {
		return fmt.Errorf("valid acl failed, arg of acl is nil")
	}

	// permission model check
	if permissionModel := acl.GetPm(); permissionModel != nil {
		permissionRule := permissionModel.GetRule()
		akSets := acl.GetAkSets()
		aksWeight := acl.GetAksWeight()
		if akSets == nil && aksWeight == nil {
			return fmt.Errorf("invoke NewAccount failed, permission model is not valid")
		}
		// aks limitation check
		if permissionRule == protos.PermissionRule_SIGN_THRESHOLD {
			if aksWeight == nil || len(aksWeight) > base.GetAkLimit() {
				return fmt.Errorf("valid acl failed, aksWeight is nil or size of aksWeight is very big")
			}
		} else if permissionRule == protos.PermissionRule_SIGN_AKSET {
			if akSets != nil {
				sets := akSets.GetSets()
				if sets == nil || len(sets) > base.GetAkLimit() {
					return fmt.Errorf("valid acl failed, Sets is nil or size of Sets is very big")
				}
			} else {
				return fmt.Errorf("valid acl failed, akSets is nil")
			}
		} else {
			return fmt.Errorf("valid acl failed, permission model is not found")
		}
	} else {
		return fmt.Errorf("valid acl failed, lack of argument of permission model")
	}

	return nil
}
