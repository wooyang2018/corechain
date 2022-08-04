package govern

import "github.com/wooyang2018/corechain/protos"

type GovManager interface {
	GetGovTokenBalance(accountName string) (*protos.GovernTokenBalance, error)
	DetermineGovTokenIfInitialized() (bool, error)
}
