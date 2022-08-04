package reader

import (
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/engines/base"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/protos"
)

type ContractReader interface {
	// 查询该链合约统计数据
	QueryContractStatData() (*protos.ContractStatData, error)
	// 查询账户下合约状态
	GetAccountContracts(account string) ([]*protos.ContractStatus, error)
	// 查询地址下合约状态
	GetAddressContracts(addr string, needContent bool) (map[string][]*protos.ContractStatus, error)
	// 查询地址下账户
	GetAccountByAK(addr string) ([]string, error)
	// 查询合约账户ACL
	QueryAccountACL(account string) (*protos.Acl, error)
	// 查询合约方法ACL
	QueryContractMethodACL(contract, method string) (*protos.Acl, error)
	// 查询账户治理代币余额
	QueryAccountGovernTokenBalance(account string) (*protos.GovernTokenBalance, error)
}

type contractReader struct {
	chainCtx *base.ChainCtx
	baseCtx  xctx.Context
	log      logger.Logger
}

func NewContractReader(chainCtx *base.ChainCtx, baseCtx xctx.Context) ContractReader {
	if chainCtx == nil || baseCtx == nil {
		return nil
	}

	reader := &contractReader{
		chainCtx: chainCtx,
		baseCtx:  baseCtx,
		log:      baseCtx.GetLog(),
	}

	return reader
}

func (t *contractReader) QueryContractStatData() (*protos.ContractStatData, error) {
	contractStatData, err := t.chainCtx.State.QueryContractStatData()
	if err != nil {
		return nil, base.CastError(err)
	}

	return contractStatData, nil
}

func (t *contractReader) GetAccountContracts(account string) ([]*protos.ContractStatus, error) {
	contracts, err := t.chainCtx.State.GetAccountContracts(account)
	if err != nil {
		return nil, base.CastError(err)
	}

	contractStatusList := make([]*protos.ContractStatus, 0, len(contracts))
	for _, contractName := range contracts {
		status, err := t.chainCtx.State.GetContractStatus(contractName)
		if err != nil {
			t.log.Warn("get contract status error", "err", err)
			return nil, base.CastError(err)
		}

		contractStatusList = append(contractStatusList, status)
	}

	return contractStatusList, nil
}

func (t *contractReader) GetAddressContracts(address string,
	needContent bool) (map[string][]*protos.ContractStatus, error) {

	accounts, err := t.GetAccountByAK(address)
	if err != nil {
		return nil, base.CastError(err)
	}

	out := make(map[string][]*protos.ContractStatus, len(accounts))
	for _, account := range accounts {
		contractStatusList, err := t.GetAccountContracts(account)
		if err != nil {
			t.log.Warn("get account contracts error", "err", err, "account", account)
			continue
		}

		out[account] = contractStatusList
	}

	return out, nil
}

func (t *contractReader) GetAccountByAK(address string) ([]string, error) {
	accounts, err := t.chainCtx.State.QueryAccountContainAK(address)
	if err != nil {
		return nil, base.CastError(err)
	}

	return accounts, nil
}

func (t *contractReader) QueryAccountACL(account string) (*protos.Acl, error) {
	acl, err := t.chainCtx.State.QueryAccountACL(account)
	if err != nil {
		return nil, base.CastError(err)
	}

	return acl, nil
}

func (t *contractReader) QueryContractMethodACL(contract, method string) (*protos.Acl, error) {
	acl, err := t.chainCtx.State.QueryContractMethodACL(contract, method)
	if err != nil {
		return nil, base.CastError(err)
	}

	return acl, nil
}

func (t *contractReader) QueryAccountGovernTokenBalance(account string) (*protos.GovernTokenBalance, error) {
	amount, err := t.chainCtx.State.QueryAccountGovernTokenBalance(account)
	if err != nil {
		return nil, base.CastError(err)
	}

	return amount, nil
}
