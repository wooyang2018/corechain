package single

import (
	"encoding/json"
	"sync"
)

type ValidatorsInfo struct {
	Validators []string `json:"validators"`
}

type SingleStatus struct {
	startHeight int64
	mutex       sync.RWMutex
	newHeight   int64
	index       int
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
