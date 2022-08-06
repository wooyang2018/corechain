package xpos

import (
	"encoding/json"
	"testing"
)

func TestGetCurrentValidatorsInfo(t *testing.T) {
	cstr := getXPOSConsensusConf()
	tdposCfg, err := buildConfigs([]byte(cstr))
	if err != nil {
		t.Error("Config unmarshal err", "err", err)
	}
	cctx, err := prepare(getXPOSConsensusConf())
	if err != nil {
		t.Error("prepare error", "error", err)
		return
	}
	s := NewSchedule(tdposCfg, cctx.XLog, cctx.Ledger, 1)
	status := TdposStatus{
		Version:     1,
		StartHeight: 1,
		Index:       0,
		election:    s,
	}
	b := status.GetCurrentValidatorsInfo()
	var addrs ValidatorsInfo
	if err := json.Unmarshal(b, &addrs); err != nil {
		t.Error("GetCurrentValidatorsInfo error", "error", err)
	}
}
