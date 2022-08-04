package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config is the envconfig of p2p server. Attention, envconfig of dht are not expose
type NetConf struct {
	// Module is the name of p2p module plugin
	Module string `yaml:"module,omitempty"`
	// Port the p2p network listened for p2p
	Port int32 `yaml:"port,omitempty"`
	// Address multiaddr string, /ip4/127.0.0.1/tcp/8080
	Address string `yaml:"address,omitempty"`
	// keyPath is the node private key path
	KeyPath string `yaml:"keyPath,omitempty"`
	// isNat envconfig whether the node use NAT manager
	IsNat bool `yaml:"isNat,omitempty"`
	// isHidden envconfig whether the node can be found
	IsHidden bool `yaml:"isHidden,omitempty"`
	// bootNodes envconfig the bootNodes the node to connect
	BootNodes []string `yaml:"bootNodes,omitempty"`
	// staticNodes envconfig the nodes which you trust
	StaticNodes map[string][]string `yaml:"staticNodes,omitempty"`
	// isBroadCast envconfig whether broadcast to all StaticNodes
	IsBroadCast bool `yaml:"isBroadCast,omitempty"`
	// maxStreamLimits envconfig the max stream num
	MaxStreamLimits int32 `yaml:"maxStreamLimits,omitempty"`
	// maxMessageSize envconfig the max message size
	MaxMessageSize int64 `yaml:"maxMessageSize,omitempty"`
	// timeout envconfig the timeout of Request with response
	Timeout int64 `yaml:"timeout,omitempty"`
	// StreamIPLimitSize set the limitation size for same ip
	StreamIPLimitSize int64 `yaml:"streamIPLimitSize,omitempty"`
	// MaxBroadcastPeers limit the number of common peers in a broadcast,
	// this number do not include MaxBroadcastCorePeers.
	MaxBroadcastPeers int `yaml:"maxBroadcastPeers,omitempty"`
	// isTls envconfig the node use tls secure transparent
	IsTls bool `yaml:"isTls,omitempty"`
	// ServiceName
	ServiceName string `yaml:"serviceName,omitempty"`
}

func LoadP2PConf(cfgFile string) (*NetConf, error) {
	cfg := GetDefP2PConf()
	err := cfg.loadConf(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load p2p envconfig failed.err:%s", err)
	}

	return cfg, nil
}

func GetDefP2PConf() *NetConf {
	return &NetConf{
		Module:          DefaultModule,
		Port:            DefaultPort,
		Address:         DefaultAddress,
		KeyPath:         DefaultNetKeyPath,
		IsNat:           DefaultNetIsNat,
		IsTls:           DefaultNetIsTls,
		IsHidden:        DefaultNetIsHidden,
		MaxStreamLimits: DefaultMaxStreamLimits,
		MaxMessageSize:  DefaultMaxMessageSize,
		Timeout:         DefaultTimeout,
		// default stream ip limit size
		StreamIPLimitSize: DefaultStreamIPLimitSize,
		MaxBroadcastPeers: DefaultMaxBroadcastPeers,
		StaticNodes:       make(map[string][]string),
		ServiceName:       DefaultServiceName,
		IsBroadCast:       DefaultIsBroadCast,
	}
}

func (t *NetConf) loadConf(cfgFile string) error {
	if cfgFile == "" {
		return fmt.Errorf("envconfig file set error.path:%s", cfgFile)
	}

	viperObj := viper.New()
	viperObj.SetConfigFile(cfgFile)
	err := viperObj.ReadInConfig()
	if err != nil {
		return fmt.Errorf("read envconfig failed.path:%s,err:%v", cfgFile, err)
	}

	if err = viperObj.Unmarshal(t); err != nil {
		return fmt.Errorf("unmatshal envconfig failed.path:%s,err:%v", cfgFile, err)
	}

	return nil
}
