package rule

import (
	"errors"

	"github.com/wooyang2018/corechain/permission/ptree"
	"github.com/wooyang2018/corechain/protos"
)

var InvalidErr = errors.New("Validate: Invalid Param")

// ACLValidator interface defines base interface for ACL Validator
// Validator only validate account/ak with 1~2 level height
type ACLValidator interface {
	Validate(pnode *ptree.PermNode) (bool, error)
}

// ACLValidatorFactory create ACLValidator for specified permission model
type ACLValidatorFactory struct {
}

// GetACLValidator returns ACLValidator for specified permission model
func (vf *ACLValidatorFactory) GetACLValidator(pr protos.PermissionRule) (ACLValidator, error) {
	switch pr {
	case protos.PermissionRule_SIGN_THRESHOLD:
		return NewThresholdValidator(), nil
	case protos.PermissionRule_SIGN_AKSET:
		return NewAKSetsValidator(), nil
	case protos.PermissionRule_SIGN_RATE:
		return vf.notImplementedValidator()
	case protos.PermissionRule_SIGN_SUM:
		return vf.notImplementedValidator()
	case protos.PermissionRule_CA_SERVER:
		return vf.notImplementedValidator()
	case protos.PermissionRule_COMMUNITY_VOTE:
		return vf.notImplementedValidator()
	}
	return nil, errors.New("Unknown permission rule")
}

// notImplementedValidator return error for not implemented validator of PermissionRule
func (vf *ACLValidatorFactory) notImplementedValidator() (ACLValidator, error) {
	return nil, errors.New("This permission rule is not implemented")
}
