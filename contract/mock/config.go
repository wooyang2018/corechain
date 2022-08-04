package mock

import (
	"github.com/wooyang2018/corechain/contract"
	"github.com/wooyang2018/corechain/logger"
)

func GetMockContractConfig() *contract.ContractConfig {
	var log, _ = logger.NewLogger("", "contract")
	return &contract.ContractConfig{
		EnableUpgrade: true,
		Xkernel: contract.XkernelConfig{
			Enable: true,
			Driver: "default",
		},
		Native: contract.NativeConfig{
			Enable: true,
			Driver: "native",
		},
		LogDriver: log,
	}

}
