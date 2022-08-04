package bridge

import (
	"github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/protos"
)

func eventsResourceUsed(events []*protos.ContractEvent) base.Limits {
	var size int64
	for _, event := range events {
		size += int64(len(event.Contract) + len(event.Name) + len(event.Body))
	}
	return base.Limits{
		Disk: size,
	}
}
