package rule

import (
	"github.com/wooyang2018/corechain/protos"
)

// ThresholdValidator is Valiator for Threshold permission model
type ThresholdValidator struct{}

// NewThresholdValidator return instance of ThresholdValidator
func NewThresholdValidator() *ThresholdValidator {
	return &ThresholdValidator{}
}

// Validate implements the interface of ACLValidator
func (tv *ThresholdValidator) Validate(pnode *PermNode) (bool, error) {
	var weightSum float64

	if pnode == nil {
		return false, InvalidErr
	}

	weightSum = 0
	for _, node := range pnode.Children {
		// the child account/ak must be passed the validation before
		if node.Status != Success {
			continue
		}

		// the child account/ak should be member in ACL list
		weight := tv.findWeightInACL(node.Name, pnode.ACL)
		weightSum += weight
	}
	return (weightSum >= pnode.ACL.Pm.AcceptValue), nil
}

func (tv *ThresholdValidator) findWeightInACL(name string, acl *protos.Acl) float64 {
	if acl == nil || acl.Pm == nil || len(acl.AksWeight) == 0 {
		return 0
	}
	weight, ok := acl.AksWeight[name]
	if !ok {
		return 0
	}
	return weight
}
