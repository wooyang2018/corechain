package bridge

import (
	"github.com/wooyang2018/corechain/contract"
	"github.com/wooyang2018/corechain/protos"
)

func eventsResourceUsed(events []*protos.ContractEvent) contract.Limits {
	var size int64
	for _, event := range events {
		size += int64(len(event.Contract) + len(event.Name) + len(event.Body))
	}
	return contract.Limits{
		Disk: size,
	}
}
