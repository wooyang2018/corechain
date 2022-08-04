package config

// default settings
const (
	DefaultModule            = "network"
	DefaultPort              = 47101 // p2p port
	DefaultAddress           = "/ip4/127.0.0.1/tcp/47101"
	DefaultNetKeyPath        = "netkeys" // node private key path
	DefaultNetIsNat          = true      // use NAT
	DefaultNetIsTls          = false     // use tls secure transport
	DefaultNetIsHidden       = false
	DefaultMaxStreamLimits   = 1024
	DefaultMaxMessageSize    = 128
	DefaultTimeout           = 30
	DefaultStreamIPLimitSize = 10
	DefaultMaxBroadcastPeers = 20
	DefaultServiceName       = "localhost"
	DefaultIsBroadCast       = true
)
