package xpoa

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/wooyang2018/corechain/consensus/base"
	contractBase "github.com/wooyang2018/corechain/contract/base"
)

// methodEditValidates 候选人变更，替代原三代合约的add_validates/delete_validates/change_validates三个操作方法
// Args: validates::候选人钱包地址
func (x *XPOAConsensus) methodEditValidates(contractCtx contractBase.KContext) (*contractBase.Response, error) {
	// 核查变更候选人合约参数有效性
	txArgs := contractCtx.Args()
	// 1. 核查发起者的权限
	aks := make(map[string]float64)
	if err := json.Unmarshal(txArgs["aksWeight"], &aks); err != nil {
		return base.NewContractBadResponse("invalid acl: unmarshal err."), err
	}
	totalBytes := txArgs["rule"]
	totalStr := string(totalBytes)
	total, err := strconv.ParseInt(totalStr, 10, 32)
	if total != 1 || err != nil { // 目前必须是阈值模型
		return base.NewContractBadResponse("invalid acl: rule should eq 1."), err
	}
	acceptBytes := txArgs["acceptValue"]
	acceptStr := string(acceptBytes)
	acceptValue, err := strconv.ParseFloat(acceptStr, 64)
	if err != nil {
		return base.NewContractBadResponse("invalid acl: pls check accept value."), err
	}

	curValiBytes, err := contractCtx.Get(x.election.bindContractBucket,
		[]byte(fmt.Sprintf("%d_%s", x.election.consensusVersion, validateKeys)))
	curVali, err := func() ([]string, error) {
		if err != nil || curValiBytes == nil {
			return x.election.initValidators, nil
		}
		var curValiKey ProposerInfo
		err = json.Unmarshal(curValiBytes, &curValiKey)
		if err != nil {
			x.log.Error("Unmarshal error")
			return nil, err
		}
		return curValiKey.Address, nil
	}()
	if err != nil {
		return base.NewContractBadResponse(err.Error()), err
	}
	if !x.isAuthAddress(curVali, aks, acceptValue, x.election.enableBFT) {
		return base.NewContractBadResponse(aclErr.Error()), aclErr
	}

	// 2. 检查desc参数权限
	validatesBytes := txArgs["validates"]
	validatesAddrs := string(validatesBytes)
	if validatesAddrs == "" {
		return base.NewContractBadResponse(targetParamErr.Error()), targetParamErr
	}
	validators := strings.Split(validatesAddrs, ";")
	rawV := &ProposerInfo{
		Address: validators,
	}
	rawBytes, err := json.Marshal(rawV)
	if err != nil {
		return base.NewContractErrResponse(err.Error()), err
	}
	if err := contractCtx.Put(x.election.bindContractBucket,
		[]byte(fmt.Sprintf("%d_%s", x.election.consensusVersion, validateKeys)), rawBytes); err != nil {
		return base.NewContractErrResponse(err.Error()), err
	}
	delta := contractBase.Limits{
		XFee: FEE,
	}
	contractCtx.AddResourceUsed(delta)
	return base.NewContractOKResponse(rawBytes), nil
}

// methodGetValidates 候选人获取
// Return: validates::候选人钱包地址
func (x *XPOAConsensus) methodGetValidates(contractCtx contractBase.KContext) (*contractBase.Response, error) {
	var jsonBytes []byte
	validatesBytes, err := contractCtx.Get(x.election.bindContractBucket,
		[]byte(fmt.Sprintf("%d_%s", x.election.consensusVersion, validateKeys)))
	if err != nil {
		returnV := map[string][]string{
			"validators": x.election.initValidators,
		}
		jsonBytes, err = json.Marshal(returnV)
		if err != nil {
			return base.NewContractErrResponse(err.Error()), err
		}
	} else {
		jsonBytes = validatesBytes
	}
	delta := contractBase.Limits{
		XFee: FEE / 1000,
	}
	contractCtx.AddResourceUsed(delta)
	return base.NewContractOKResponse(jsonBytes), nil
}

// isAuthAddress 判断输入aks是否能在贪心下仍能满足签名数量>33%(Chained-BFT装载) or 50%(一般情况)
func (x *XPOAConsensus) isAuthAddress(validators []string, aks map[string]float64, threshold float64, enableBFT bool) bool {
	// 0. 是否是单个候选人
	if len(validators) == 1 {
		weight, ok := aks[validators[0]]
		if !ok {
			return false
		}
		return weight >= threshold
	}
	// 1. 判断aks中的地址是否是当前集合地址
	for addr, _ := range aks {
		if !Find(addr, validators) {
			return false
		}
	}
	// 2. 判断贪心下签名集合数目仍满足要求
	var s aksSlice
	for k, v := range aks {
		s = append(s, aksItem{
			Address: k,
			Weight:  v,
		})
	}
	sort.Stable(s)
	greedyCount := 0
	sum := threshold
	for i := 0; i < len(aks); i++ {
		if sum <= 0 {
			break
		}
		sum -= s[i].Weight
		greedyCount++
	}
	if !enableBFT {
		return greedyCount >= len(validators)/2+1
	}
	return CalFault(int64(greedyCount), int64(len(validators)))
}
