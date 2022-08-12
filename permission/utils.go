package permission

import (
	"errors"
	"fmt"
	"strings"

	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/permission/base"
	"github.com/wooyang2018/corechain/permission/rule"
	"github.com/wooyang2018/corechain/protos"
)

func IdentifyAK(akuri string, sign *protos.SignatureInfo, msg []byte) (bool, error) {
	if sign == nil {
		return false, errors.New("sign is nil")
	}
	akpath := strings.Split(akuri, "/")
	if len(akpath) < 1 {
		return false, errors.New("Invalid address")
	}
	ak := akpath[len(akpath)-1]
	return VerifySign(ak, sign, msg)
}

func IdentifyAccount(aclMgr base.AclManager, account string, aksuri []string) (bool, error) {
	// aks and signs could have zero length for permission rule Null
	if aclMgr == nil {
		return false, fmt.Errorf("Invalid Param, aclMgr=%v", aclMgr)
	}

	// build perm tree
	pnode, err := rule.BuildAccountPermTree(aclMgr, account, aksuri)
	if err != nil {
		return false, err
	}

	return validatePermTree(pnode, true)
}

func CheckContractMethodPerm(aclMgr base.AclManager, aksuri []string,
	contractName, methodName string) (bool, error) {

	// aks and signs could have zero length for permission rule Null
	if aclMgr == nil {
		return false, fmt.Errorf("Invalid Param, aclMgr=%v", aclMgr)
	}

	// build perm tree
	pnode, err := rule.BuildMethodPermTree(aclMgr, contractName, methodName, aksuri)
	if err != nil {
		return false, err
	}

	// validate perm tree
	return validatePermTree(pnode, false)
}

func validatePermTree(root *rule.PermNode, isAccount bool) (bool, error) {
	if root == nil {
		return false, errors.New("Root is null")
	}

	// get BFS list of perm tree
	plist, err := rule.GetPermTreeList(root)
	if err != nil {
		return false, err
	}
	listlen := len(plist)
	vf := &rule.ACLValidatorFactory{}

	// reverse travel the perm tree
	for i := listlen - 1; i >= 0; i-- {
		pnode := plist[i]
		nameCheck := base.IsAccount(pnode.Name)
		// 0 means AK, 1 means Account, otherwise invalid
		if nameCheck < 0 || nameCheck > 1 {
			return false, errors.New("Invalid account/ak name")
		}

		// for non-account perm tree, the root node is not account name
		if i == 0 && !isAccount {
			nameCheck = 1
		}

		checkResult := false
		if nameCheck == 0 {
			// current node is AK, signature should be validated before
			checkResult = true
		} else if nameCheck == 1 {
			// current node is Account, so validation using ACLValidator
			if pnode.ACL == nil {
				// empty ACL means everyone could pass ACL validation
				checkResult = true
			} else {
				if pnode.ACL.Pm == nil {
					return false, errors.New("Acl has empty Pm field")
				}

				// get ACLValidator by ACL type
				validator, err := vf.GetACLValidator(pnode.ACL.Pm.Rule)
				if err != nil {
					return false, err
				}
				checkResult, err = validator.Validate(pnode)
				if err != nil {
					return false, err
				}
			}
		}

		// set validation status
		if checkResult {
			pnode.Status = rule.Success
		} else {
			pnode.Status = rule.Failed
		}
	}
	return (root.Status == rule.Success), nil
}

// GetAccountACL return account acl
func GetAccountACL(aclMgr base.AclManager, account string) (*protos.Acl, error) {
	return aclMgr.GetAccountACL(account)
}

// GetContractMethodACL return contract method acl
func GetContractMethodACL(aclMgr base.AclManager, contractName, methodName string) (*protos.Acl, error) {
	return aclMgr.GetContractMethodACL(contractName, methodName)
}

func VerifySign(ak string, si *protos.SignatureInfo, data []byte) (bool, error) {
	bytespk := []byte(si.PublicKey)
	xcc, err := cryptoClient.CreateCryptoClientFromJSONPublicKey(bytespk)
	if err != nil {
		return false, err
	}

	ecdsaKey, err := xcc.GetEcdsaPublicKeyFromJsonStr(string(bytespk[:]))
	if err != nil {
		return false, err
	}

	isMatch, _ := xcc.VerifyAddressUsingPublicKey(ak, ecdsaKey)
	if !isMatch {
		return false, errors.New("address and public key not match")
	}

	return xcc.VerifyECDSA(ecdsaKey, si.Sign, data)
}
