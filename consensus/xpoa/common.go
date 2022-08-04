package xpoa

import (
	"encoding/json"
	"errors"
	"strconv"
)

var (
	MinerSelectErr   = errors.New("Node isn't a miner, calculate error.")
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
	InvalidQC        = errors.New("QC struct is invalid.")
	targetParamErr   = errors.New("Target paramters are invalid, please check them.")
	tooLowHeight     = errors.New("The height should be higher than 3.")
	aclErr           = errors.New("Xpoa needs valid acl account.")
	scheduleErr      = errors.New("minerScheduling overflow")
)

const (
	xpoaBucket           = "$xpoa"
	poaBucket            = "$poa"
	validateKeys         = "validates"
	contractGetValidates = "getValidates"
	contractEditValidate = "editValidates"

	FEE          = 1000
	MAXSLEEPTIME = 1000
	MAXMAPSIZE   = 1000
)

type XPOAConfig struct {
	// 每个候选人每轮出块个数
	BlockNum int64 `json:"block_num"`
	// 单位为毫秒
	Period       int64           `json:"period"`
	InitProposer ProposerInfo    `json:"init_proposer"`
	EnableBFT    map[string]bool `json:"bft_config,omitempty"`
}

type ProposerInfo struct {
	Address []string `json:"address"`
}

// LoadValidatorsMultiInfo xpoa 格式为 { "address": [$ADDR_STRING...] }
func loadValidatorsMultiInfo(res []byte) ([]string, error) {
	if res == nil {
		return nil, NotValidContract
	}
	contractInfo := ProposerInfo{}
	if err := json.Unmarshal(res, &contractInfo); err != nil {
		return nil, err
	}
	return contractInfo.Address, nil
}

func Find(a string, t []string) bool {
	for _, v := range t {
		if a != v {
			continue
		}
		return true
	}
	return false
}

//CalFault 根据3f+1, 计算最大恶意节点数
func CalFault(input, sum int64) bool {
	f := (sum - 1) / 3
	if f < 0 {
		return false
	}
	if f == 0 {
		return input >= sum/2+1
	}
	return input >= (sum-f)/2+1
}

// aksItem 每个地址每一轮的总票数
type aksItem struct {
	Address string
	Weight  float64
}

type aksSlice []aksItem

func (a aksSlice) Len() int {
	return len(a)
}

func (a aksSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a aksSlice) Less(i, j int) bool {
	if a[j].Weight == a[i].Weight {
		return a[j].Address < a[i].Address
	}
	return a[j].Weight < a[i].Weight
}

type StringConfig struct {
	Version string `json:"version,omitempty"`
}

type IntConfig struct {
	Version int64 `json:"version,omitempty"`
}

// ParseVersion 支持string格式和int格式的version
func ParseVersion(cfg string) (int64, error) {
	intVersion := IntConfig{}
	if err := json.Unmarshal([]byte(cfg), &intVersion); err == nil {
		return intVersion.Version, nil
	}
	strVersion := StringConfig{}
	if err := json.Unmarshal([]byte(cfg), &strVersion); err != nil {
		return 0, err
	}
	version, err := strconv.ParseInt(strVersion.Version, 10, 64)
	if err != nil {
		return 0, err
	}
	return version, nil
}
