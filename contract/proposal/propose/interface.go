package propose

import (
	"github.com/wooyang2018/corechain/protos"
)

type ProposeManager interface {
	GetProposalByID(proposalID string) (*protos.Proposal, error)
}
