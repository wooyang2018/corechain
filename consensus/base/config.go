package base

// ConsensusConfig 特定共识的字段标示
type ConsensusConfig struct {
	// 本次共识的类型名称
	ConsensusName string `json:"name"`
	// 本次共识专属属性
	Config string `json:"config"`
	// 本次共识的起始高度
	StartHeight int64 `json:"height,omitempty"`
	// 本次共识在consensus slice中的index
	Index int `json:"index,omitempty"`
}
