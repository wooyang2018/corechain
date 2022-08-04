package ledger

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"github.com/wooyang2018/corechain/common/cache"
)

// awardCacheSize system award cache, in avoid of double computing
const awardCacheSize = 1000

// GenesisBlock genesis block data structure
type GenesisBlock struct {
	config     *GenesisConfig
	awardCache *cache.LRUCache
}

// NewGenesisBlock new a genesis block
func NewGenesisBlock(genesisCfg []byte) (*GenesisBlock, error) {
	if len(genesisCfg) < 1 {
		return nil, fmt.Errorf("genesis config is empty")
	}

	// 加载配置
	config := &GenesisConfig{}
	jsErr := json.Unmarshal(genesisCfg, config)
	if jsErr != nil {
		return nil, jsErr
	}
	if config.NoFee {
		config.Award = "0"
		config.NewAccountResourceAmount = 0
		// nofee场景下，不需要原生代币xuper
		// 但是治理代币，会从此配置中进行初始代币发行，故而保留config.Predistribution内容
		//config.Predistribution = []struct {
		//	Address string `json:"address"`
		//	Quota   string `json:"quota"`
		//}{}
		config.GasPrice.CpuRate = 0
		config.GasPrice.DiskRate = 0
		config.GasPrice.MemRate = 0
		config.GasPrice.XfeeRate = 0
	}

	gb := &GenesisBlock{
		awardCache: cache.NewLRUCache(awardCacheSize),
		config:     config,
	}

	return gb, nil
}

// GetConfig get config of genesis block
func (gb *GenesisBlock) GetConfig() *GenesisConfig {
	return gb.config
}

// CalcAward calc system award by block height
func (gb *GenesisBlock) CalcAward(blockHeight int64) *big.Int {
	award := big.NewInt(0)
	award.SetString(gb.config.Award, 10)
	if gb.config.AwardDecay.HeightGap == 0 { //无衰减策略
		return award
	}
	period := blockHeight / gb.config.AwardDecay.HeightGap
	if awardRemember, ok := gb.awardCache.Get(period); ok {
		return awardRemember.(*big.Int) //加个记忆，避免每次都重新算
	}
	var realAward = float64(award.Int64())
	for i := int64(0); i < period; i++ { //等比衰减
		realAward = realAward * gb.config.AwardDecay.Ratio
	}
	N := int64(math.Round(realAward)) //四舍五入
	award.SetInt64(N)
	gb.awardCache.Add(period, award)
	return award
}
