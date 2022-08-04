package rule

import (
	"github.com/wooyang2018/corechain/permission/ptree"
	"github.com/wooyang2018/corechain/protos"
)

//TODO 我还是没能理解这个的验证策略是什么
// AKSetsValidator is Valiator for AkSets permission model
type AKSetsValidator struct{}

// NewAKSetsValidator return instance of AKSetsValidator
func NewAKSetsValidator() *AKSetsValidator {
	return &AKSetsValidator{}
}

// Validate implements the interface of ACLValidator
func (asv *AKSetsValidator) Validate(pnode *ptree.PermNode) (bool, error) {
	expResult := false
	if pnode == nil {
		return false, InvalidErr
	}

	// empty ACL means everyone can pass the validation
	if pnode.ACL == nil {
		return true, nil
	}

	// empty or null AkSets means no one can pass the validation
	if pnode.ACL.AkSets == nil || len(pnode.ACL.AkSets.Sets) == 0 {
		return false, nil
	}

	// AkSets.Expression is not supported now, only support default expression:
	// 1. AkSet: a set is valid only if all aks pass signature verification
	// 2. AkSets: an AkSets is valid only if at least one AkSet pass signature verification
	for _, set := range pnode.ACL.AkSets.Sets {
		if isValid := asv.validateAkSet(set, pnode.Children); isValid {
			expResult = true
			break
		}
	}

	return expResult, nil
}

// validateAkSet validate single AkSet 验证是否每个人都成功了
func (asv *AKSetsValidator) validateAkSet(set *protos.AkSet, signedAks []*ptree.PermNode) bool {
	// empty set or empty signature means validate failed
	if len(set.Aks) == 0 || len(signedAks) == 0 {
		return false
	}

	isValid := true
	for _, ak := range set.Aks {
		node := asv.findAkInNodeList(ak, signedAks)
		if node == nil || node.Status != ptree.Success {
			// found one ak without valid signature, this set validate failed
			isValid = false
			break
		}
	}
	return isValid
}

// findAkInNodeList find permnode with specified name
func (asv *AKSetsValidator) findAkInNodeList(name string, signedAks []*ptree.PermNode) *ptree.PermNode {
	var pnode *ptree.PermNode
	pnode = nil
	//TODO 但凡写一个Map也不至于要遍历啊
	for _, node := range signedAks {
		if node.Name == name {
			pnode = node
			break
		}
	}

	return pnode
}
