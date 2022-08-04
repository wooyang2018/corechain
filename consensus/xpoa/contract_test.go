package xpoa

import (
	"encoding/json"
	"testing"

	cmock "github.com/wooyang2018/corechain/consensus/mock"
)

func NewEditArgs() map[string][]byte {
	a := make(map[string][]byte)
	a["validates"] = []byte(`{
		"validates":"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN;WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT;akf7qunmeaqb51Wu418d6TyPKp4jdLdpV"
	}`)
	a["rule"] = []byte("1")
	a["acceptValue"] = []byte("0.600")
	aks := map[string]float64{
		"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN": 0.5,
		"WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT": 0.5,
	}
	a["aksWeight"], _ = json.Marshal(&aks)
	return a
}

func TestMethodEditValidates(t *testing.T) {
	cctx, err := prepare(getXPOAConsensusConf())
	if err != nil {
		t.Fatal("prepare error", "error", err)
	}
	i := NewXPOAConsensus(*cctx, getConfig(getXPOAConsensusConf()))
	xpoa, ok := i.(*XPOAConsensus)
	if !ok {
		t.Fatal("transfer err.")
	}
	fakeCtx := cmock.NewFakeKContext(NewEditArgs(), make(map[string]map[string][]byte))
	//第一次Get应当返回空Body
	r, err := xpoa.methodGetValidates(fakeCtx)
	if err != nil || r.Body != nil {
		t.Fatal("methodGetValidates error", "error", err, "r", r)
	}
	//第二次更新候选人
	r, err = xpoa.methodEditValidates(fakeCtx)
	if err != nil {
		t.Fatal("methodEditValidates error", "error", err, "r", r)
	}
	//第三次Get应当返回更新后的候选人集合
	r, err = xpoa.methodGetValidates(fakeCtx)
	if err != nil || r.Body == nil {
		t.Fatal("methodGetValidates error", "error", err, "r", r)
	}
	t.Log(string(r.Body))
}

func TestIsAuthAddress(t *testing.T) {
	aks := map[string]float64{
		"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN": 0.5,
		"WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT": 0.6,
	}
	cctx, err := prepare(getXPOAConsensusConf())
	if err != nil {
		t.Fatal("prepare error", "error", err)
	}
	i := NewXPOAConsensus(*cctx, getConfig(getXPOAConsensusConf()))
	xpoa, ok := i.(*XPOAConsensus)
	if !ok {
		t.Fatal("transfer err.")
	}
	v1 := []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN", "WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	if !xpoa.isAuthAddress(v1, aks, 0.7, true) {
		t.Fatal("isAuthAddress err.")
	}
	v2 := []string{"dpzuVdosQrF2kmzumhVeFQZa1aYcdgFpN"}
	if xpoa.isAuthAddress(v2, aks, 0.6, true) {
		t.Fatal("isAuthAddress err.")
	}
	v3 := []string{"WNWk3ekXeM5M2232dY2uCJmEqWhfQiDYT"}
	if !xpoa.isAuthAddress(v3, aks, 0.6, true) {
		t.Fatal("isAuthAddress err.")
	}
}
