package single

import (
	"encoding/json"
	"sync"
)

type ValidatorsInfo struct {
	Validators []string `json:"validators"`
}

type SingleStatus struct {
	startHeight int64 //共识开始的高度
	mutex       sync.RWMutex
	newHeight   int64 //当前的Term
	index       int   //step共识的索引
	config      *SingleConfig
}

func (s *SingleStatus) GetVersion() int64 {
	return s.config.Version
}

func (s *SingleStatus) GetConsensusBeginInfo() int64 {
	return s.startHeight
}

func (s *SingleStatus) GetStepConsensusIndex() int {
	return s.index
}

func (s *SingleStatus) GetConsensusName() string {
	return "single"
}

func (s *SingleStatus) GetCurrentTerm() int64 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.newHeight
}

func (s *SingleStatus) GetCurrentValidatorsInfo() []byte {
	miner := ValidatorsInfo{
		Validators: []string{s.config.Miner},
	}
	m, _ := json.Marshal(miner)
	return m
}
