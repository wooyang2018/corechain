package propose

import (
	"encoding/json"
	"fmt"

	"github.com/wooyang2018/corechain/contract/proposal/utils"
	"github.com/wooyang2018/corechain/protos"
)

// Manager manages all timer releated data, providing read/write interface
type Manager struct {
	Ctx *ProposeCtx
}

// NewTimerManager create instance of ProposeManager
func NewProposeManager(ctx *ProposeCtx) (ProposeManager, error) {
	if ctx == nil || ctx.Ledger == nil || ctx.Contract == nil || ctx.BcName == "" {
		return nil, fmt.Errorf("propose ctx set error")
	}

	t := NewKernContractMethod(ctx.BcName)
	register := ctx.Contract.GetKernRegistry()
	register.RegisterKernMethod(utils.ProposalKernelContract, "Propose", t.Propose)
	register.RegisterKernMethod(utils.ProposalKernelContract, "Vote", t.Vote)
	register.RegisterKernMethod(utils.ProposalKernelContract, "Thaw", t.Thaw)
	register.RegisterKernMethod(utils.ProposalKernelContract, "CheckVoteResult", t.CheckVoteResult)
	register.RegisterKernMethod(utils.ProposalKernelContract, "Trigger", t.Trigger)
	register.RegisterKernMethod(utils.ProposalKernelContract, "Query", t.Query)

	mg := &Manager{
		Ctx: ctx,
	}

	return mg, nil
}

// GetProposalByID get proposal by proposal_id
func (mgr *Manager) GetProposalByID(proposalID string) (*protos.Proposal, error) {
	proposalBuf, err := mgr.GetObjectBySnapshot(utils.GetProposalBucket(), []byte(utils.MakeProposalKey(proposalID)))
	if err != nil {
		return nil, fmt.Errorf("query proposal failed.err:%v", err)
	}

	proposal := &utils.Proposal{}
	err = json.Unmarshal(proposalBuf, proposal)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal proposal failed. err:%v", err.Error())
	}

	// todo, 此处v.([]byte)会panic，待修复
	triggerArgs := make(map[string][]byte)
	for k, v := range proposal.Trigger.Args {
		triggerArgs[k] = v.([]byte)
	}

	triggerDesc := &protos.TriggerDesc{
		Height: proposal.Trigger.Height,
		Module: proposal.Trigger.Module,
		Method: proposal.Trigger.Method,
		Args:   triggerArgs,
	}

	proposalArgs := make(map[string][]byte)
	for k, v := range proposal.Args {
		proposalArgs[k] = v.([]byte)
	}

	proposalRes := &protos.Proposal{
		Args:       proposalArgs,
		Trigger:    triggerDesc,
		VoteAmount: proposal.VoteAmount.String(),
		Status:     protos.ProposalStatus(protos.ProposalStatus_value[proposal.Status]),
	}

	return proposalRes, nil
}

func (mgr *Manager) GetObjectBySnapshot(bucket string, object []byte) ([]byte, error) {
	// 根据tip blockid 创建快照
	reader, err := mgr.Ctx.Ledger.GetTipXMSnapshotReader()
	if err != nil {
		return nil, err
	}

	return reader.Get(bucket, object)
}
