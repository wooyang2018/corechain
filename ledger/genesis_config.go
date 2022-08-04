package ledger

import (
	"strconv"

	"github.com/wooyang2018/corechain/protos"
)

// GenesisConfig genesis block configure
type GenesisConfig struct {
	Version   string `json:"version"`
	Crypto    string `json:"crypto"`
	Kvengine  string `json:"kvengine"`
	Consensus struct {
		Type  string `json:"type"`
		Miner string `json:"miner"`
	} `json:"consensus"`
	Predistribution []struct {
		Address string `json:"address"`
		Quota   string `json:"quota"`
	}
	// max block size in MB
	MaxBlockSize string `json:"maxblocksize"`
	Period       string `json:"period"`
	NoFee        bool   `json:"nofee"`
	Award        string `json:"award"`
	AwardDecay   struct {
		HeightGap int64   `json:"height_gap"`
		Ratio     float64 `json:"ratio"`
	} `json:"award_decay"`
	GasPrice struct {
		CpuRate  int64 `json:"cpu_rate"`
		MemRate  int64 `json:"mem_rate"`
		DiskRate int64 `json:"disk_rate"`
		XfeeRate int64 `json:"xfee_rate"`
	} `json:"gas_price"`
	Decimals          string                 `json:"decimals"`
	GenesisConsensus  map[string]interface{} `json:"genesis_consensus"`
	ReservedContracts []InvokeRequest        `json:"reserved_contracts"`
	ReservedWhitelist struct {
		Account string `json:"account"`
	} `json:"reserved_whitelist"`
	ForbiddenContract InvokeRequest `json:"forbidden_contract"`
	// NewAccountResourceAmount the amount of creating a new contract account
	NewAccountResourceAmount int64 `json:"new_account_resource_amount"`
	// IrreversibleSlideWindow
	IrreversibleSlideWindow string `json:"irreversibleslidewindow"`
	// GroupChainContract
	GroupChainContract InvokeRequest `json:"group_chain_contract"`
}

// InvokeRequest define genesis reserved_contracts configure
type InvokeRequest struct {
	ModuleName   string            `json:"module_name" mapstructure:"module_name"`
	ContractName string            `json:"contract_name" mapstructure:"contract_name"`
	MethodName   string            `json:"method_name" mapstructure:"method_name"`
	Args         map[string]string `json:"args" mapstructure:"args"`
}

type Predistribution struct {
	Address string `json:"address"`
	Quota   string `json:"quota"`
}

func InvokeRequestFromJSON2Pb(jsonRequest []InvokeRequest) ([]*protos.InvokeRequest, error) {
	requestsWithPb := []*protos.InvokeRequest{}
	for _, request := range jsonRequest {
		tmpReqWithPB := &protos.InvokeRequest{
			ModuleName:   request.ModuleName,
			ContractName: request.ContractName,
			MethodName:   request.MethodName,
			Args:         make(map[string][]byte),
		}
		for k, v := range request.Args {
			tmpReqWithPB.Args[k] = []byte(v)
		}
		requestsWithPb = append(requestsWithPb, tmpReqWithPB)
	}
	return requestsWithPb, nil
}

func (rc *GenesisConfig) GetCryptoType() string {
	if rc.Crypto != "" {
		return rc.Crypto
	}

	return "default"
}

// GetIrreversibleSlideWindow get irreversible slide window
func (rc *GenesisConfig) GetIrreversibleSlideWindow() int64 {
	irreversibleSlideWindow, _ := strconv.Atoi(rc.IrreversibleSlideWindow)
	return int64(irreversibleSlideWindow)
}

// GetMaxBlockSizeInByte get max block size in Byte
func (rc *GenesisConfig) GetMaxBlockSizeInByte() (n int64) {
	maxSizeMB, _ := strconv.Atoi(rc.MaxBlockSize)
	n = int64(maxSizeMB) << 20
	return
}

// GetNewAccountResourceAmount get the resource amount of new an account
func (rc *GenesisConfig) GetNewAccountResourceAmount() int64 {
	return rc.NewAccountResourceAmount
}

// GetGenesisConsensus get consensus config of genesis block
func (rc *GenesisConfig) GetGenesisConsensus() (map[string]interface{}, error) {
	if rc.GenesisConsensus == nil {
		consCfg := map[string]interface{}{}
		consCfg["name"] = rc.Consensus.Type
		consCfg["config"] = map[string]interface{}{
			"miner":  rc.Consensus.Miner,
			"period": rc.Period,
		}
		return consCfg, nil
	}
	return rc.GenesisConsensus, nil
}

// GetReservedContract get default contract config of genesis block
func (rc *GenesisConfig) GetReservedContract() ([]*protos.InvokeRequest, error) {
	return InvokeRequestFromJSON2Pb(rc.ReservedContracts)
}

func (rc *GenesisConfig) GetForbiddenContract() ([]*protos.InvokeRequest, error) {
	return InvokeRequestFromJSON2Pb([]InvokeRequest{rc.ForbiddenContract})
}

func (rc *GenesisConfig) GetGroupChainContract() ([]*protos.InvokeRequest, error) {
	return InvokeRequestFromJSON2Pb([]InvokeRequest{rc.GroupChainContract})
}

// GetReservedWhitelistAccount return reserved whitelist account
func (rc *GenesisConfig) GetReservedWhitelistAccount() string {
	return rc.ReservedWhitelist.Account
}

// GetPredistribution return predistribution
func (rc *GenesisConfig) GetPredistribution() []Predistribution {
	return PredistributionTranslator(rc.Predistribution)
}

func PredistributionTranslator(predistribution []struct {
	Address string `json:"address"`
	Quota   string `json:"quota"`
}) []Predistribution {
	var predistributionArray []Predistribution
	for _, pd := range predistribution {
		ps := Predistribution{
			Address: pd.Address,
			Quota:   pd.Quota,
		}
		predistributionArray = append(predistributionArray, ps)
	}
	return predistributionArray
}

// GetGasPrice get gas rate for different resource(cpu, mem, disk and xfee)
func (rc *GenesisConfig) GetGasPrice() *protos.GasPrice {
	gasPrice := &protos.GasPrice{
		CpuRate:  rc.GasPrice.CpuRate,
		MemRate:  rc.GasPrice.MemRate,
		DiskRate: rc.GasPrice.DiskRate,
		XfeeRate: rc.GasPrice.XfeeRate,
	}
	return gasPrice
}
