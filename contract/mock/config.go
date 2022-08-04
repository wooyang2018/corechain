package mock

import (
	"github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/logger"
)

func GetMockContractConfig() *base.ContractConfig {
	var log, _ = logger.NewLogger("", "contract")
	return &base.ContractConfig{
		EnableUpgrade: true,
		Xkernel: base.XkernelConfig{
			Enable: true,
			Driver: "default",
		},
		Native: base.NativeConfig{
			Enable: true,
			Driver: "native",
		},
		LogDriver: log,
	}

}
