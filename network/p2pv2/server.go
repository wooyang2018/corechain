package p2pv2

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	libnet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wooyang2018/corechain/common/address"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/metrics"
	"github.com/wooyang2018/corechain/common/timer"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/network"
	"github.com/wooyang2018/corechain/network/config"
	nctx "github.com/wooyang2018/corechain/network/context"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

const (
	ServerName = "p2pv2"
)

func init() {
	network.Register(ServerName, NewP2PServer)
}

// define errors
var (
	ErrGenerateOpts     = errors.New("generate host opts error")
	ErrCreateHost       = errors.New("create host error")
	ErrCreateKadDht     = errors.New("create kad dht error")
	ErrCreateStreamPool = errors.New("create stream pool error")
	ErrCreateBootStrap  = errors.New("create bootstrap error pool error")
	ErrConnectBootStrap = errors.New("error to connect to all bootstrap")
	ErrLoadAccount      = errors.New("load account error")
	ErrStoreAccount     = errors.New("dht store account error")
	ErrConnect          = errors.New("connect all boot and static peer error")
	ErrEmptyPeer        = errors.New("empty peer")
	ErrNoResponse       = errors.New("no response")
)

// P2PServerV2 is the node in the libnet
type P2PServerV2 struct {
	ctx    *nctx.NetCtx
	log    logger.Logger
	config *config.NetConf

	id         peer.ID
	host       host.Host
	kdht       *dht.IpfsDHT
	streamPool *StreamPool
	dispatcher network.Dispatcher

	cancel      context.CancelFunc
	staticNodes map[string][]peer.ID
	// local host account
	account string
	// accounts store remote peer account: key:account => v:peer.ID
	// accounts as cache, store in dht
	accounts *cache.Cache
}

var _ network.Network = &P2PServerV2{}

// NewP2PServer create P2PServerV2 instance
func NewP2PServer() network.Network {
	return &P2PServerV2{}
}

// Init initialize p2p server using given config
func (p *P2PServerV2) Init(ctx *nctx.NetCtx) error {
	p.ctx = ctx
	p.log = ctx.GetLog()
	p.config = ctx.P2PConf

	// host
	cfg := ctx.P2PConf
	opts, err := genHostOption(ctx)
	if err != nil {
		p.log.Error("genHostOption error", "error", err)
		return ErrGenerateOpts
	}

	ho, err := libp2p.New(opts...)
	if err != nil {
		p.log.Error("Create p2p host error", "error", err)
		return ErrCreateHost
	}

	p.id = ho.ID()
	p.host = ho
	p.log.Debug("Host", "address", p.getMultiAddr(p.host.ID(), p.host.Addrs()), "config", *cfg)
	prefix := fmt.Sprintf("/%s", network.Namespace)
	// dht
	dhtOpts := []dht.Option{
		dht.Mode(dht.ModeServer),
		dht.RoutingTableRefreshPeriod(10 * time.Second),
		dht.ProtocolPrefix(protocol.ID(prefix)),
		dht.NamespacedValidator(network.Namespace, &record.NamespacedValidator{
			network.Namespace: blankValidator{},
		}),
	}
	if p.kdht, err = dht.New(ctx, ho, dhtOpts...); err != nil {
		return ErrCreateKadDht
	}

	if !cfg.IsHidden {
		if err = p.kdht.Bootstrap(ctx); err != nil {
			return ErrCreateBootStrap
		}
	}

	keyPath := ctx.EnvCfg.GenDataAbsPath(ctx.EnvCfg.KeyDir)
	p.account, err = address.LoadAddress(keyPath)
	if err != nil {
		return ErrLoadAccount
	}

	p.accounts = cache.New(cache.NoExpiration, cache.NoExpiration)
	// dispatcher
	p.dispatcher = network.NewDispatcher(ctx)

	p.streamPool, err = NewStreamPool(ctx, p.host, p.dispatcher)
	if err != nil {
		return ErrCreateStreamPool
	}

	// set static nodes
	setStaticNodes(ctx, p)

	// set broadcast peers limitation
	network.MaxBroadCast = cfg.MaxBroadcastPeers

	if err := p.connect(); err != nil {
		p.log.Error("connect all boot and static peer error")
		return ErrConnect
	}

	return nil
}

func genHostOption(ctx *nctx.NetCtx) ([]libp2p.Option, error) {
	cfg := ctx.P2PConf
	muAddr, err := multiaddr.NewMultiaddr(cfg.Address)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(muAddr),
		libp2p.EnableRelay(),
	}

	if cfg.IsNat {
		opts = append(opts, libp2p.NATPortMap())
	}

	if cfg.IsTls {
		priv, err := GetPemKeyPairFromPath(cfg.KeyPath)
		if err != nil {
			return nil, err
		}
		opts = append(opts, libp2p.Identity(priv))
		opts = append(opts, libp2p.Security(ID, NewTLS(cfg.KeyPath, cfg.ServiceName)))
	} else {
		priv, err := GetKeyPairFromPath(cfg.KeyPath)
		if err != nil {
			return nil, err
		}
		opts = append(opts, libp2p.Identity(priv))
		opts = append(opts, libp2p.Security("/secio/1.0.0", tls.New))
	}

	return opts, nil
}

func setStaticNodes(ctx *nctx.NetCtx, p *P2PServerV2) {
	cfg := ctx.P2PConf
	staticNodes := map[string][]peer.ID{}
	for bcname, peers := range cfg.StaticNodes {
		peerIDs := make([]peer.ID, 0, len(peers))
		for _, peerAddr := range peers {
			id, err := GetPeerIDByAddress(peerAddr)
			if err != nil {
				p.log.Warn("static node addr error", "peerAddr", peerAddr)
				continue
			}
			peerIDs = append(peerIDs, id)
		}
		staticNodes[bcname] = peerIDs
	}
	p.staticNodes = staticNodes
}

func (p *P2PServerV2) setKdhtValue() {
	// store: account => address
	account := GenAccountKey(p.account)
	address := p.getMultiAddr(p.host.ID(), p.host.Addrs())
	err := p.kdht.PutValue(context.Background(), account, []byte(address))
	if err != nil {
		p.log.Error("dht put account=>address value error", "error", err)
	}

	// store: peer.ID => account
	id := GenPeerIDKey(p.id)
	err = p.kdht.PutValue(context.Background(), id, []byte(p.account))
	if err != nil {
		p.log.Error("dht put id=>account value error", "error", err)
	}
}

// Start start the node
func (p *P2PServerV2) Start() {
	p.log.Debug("StartP2PServer", "address", p.host.Addrs())
	p.host.SetStreamHandler(network.ProtocolVersion, p.streamHandler)

	p.setKdhtValue()

	ctx, cancel := context.WithCancel(p.ctx)
	p.cancel = cancel

	t := time.NewTicker(time.Second * 180)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				p.log.Debug("RoutingTable", "id", p.host.ID(), "size", p.kdht.RoutingTable().Size())
				// p.kdht.RoutingTable().Print()
			}
		}
	}()
}

func (p *P2PServerV2) connect() error {
	var multiAddrs []string
	if len(p.config.BootNodes) > 0 {
		multiAddrs = append(multiAddrs, p.config.BootNodes...)
	}
	for _, ps := range p.config.StaticNodes {
		multiAddrs = append(multiAddrs, ps...)
	}
	success := p.connectPeerByAddress(multiAddrs)
	if success == 0 && len(p.config.BootNodes) != 0 {
		return ErrConnectBootStrap
	}

	return nil
}

func (p *P2PServerV2) streamHandler(netStream libnet.Stream) {
	if _, err := p.streamPool.NewStream(p.ctx, netStream); err != nil {
		p.log.Warn("new stream error")
	}
}

// Stop stop the node
func (p *P2PServerV2) Stop() {
	p.log.Info("StopP2PServer")
	p.kdht.Close()
	p.host.Close()
	if p.cancel != nil {
		p.cancel()
	}
}

// PeerID return the peer ID
func (p *P2PServerV2) PeerID() string {
	return p.id.Pretty()
}

func (p *P2PServerV2) NewSubscriber(typ protos.CoreMessage_MessageType, v interface{}, opts ...network.SubscriberOption) network.Subscriber {
	return network.NewSubscriber(p.ctx, typ, v, opts...)
}

// Register register message subscriber to handle messages
func (p *P2PServerV2) Register(sub network.Subscriber) error {
	return p.dispatcher.Register(sub)
}

// UnRegister remove message subscriber
func (p *P2PServerV2) UnRegister(sub network.Subscriber) error {
	return p.dispatcher.UnRegister(sub)
}

func (p *P2PServerV2) HandleMessage(stream network.Stream, msg *protos.CoreMessage) error {
	if p.dispatcher == nil {
		p.log.Warn("dispatcher not ready, omit", "msg", msg)
		return nil
	}

	if p.ctx.EnvCfg.MetricSwitch {
		tm := time.Now()
		defer func() {
			labels := prometheus.Labels{
				metrics.LabelBCName:      msg.GetHeader().GetBcname(),
				metrics.LabelMessageType: msg.GetHeader().GetType().String(),
			}
			metrics.NetworkMsgReceivedCounter.With(labels).Inc()
			metrics.NetworkMsgReceivedBytesCounter.With(labels).Add(float64(proto.Size(msg)))
			metrics.NetworkServerHandlingHistogram.With(labels).Observe(time.Since(tm).Seconds())
		}()
	}

	if err := p.dispatcher.Dispatch(msg, stream); err != nil {
		p.log.Warn("handle new message dispatch error", "log_id", msg.GetHeader().GetLogid(),
			"type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom(), "error", err)
		return nil // not return err
	}

	return nil
}

func (p *P2PServerV2) Context() *nctx.NetCtx {
	return p.ctx
}

func (p *P2PServerV2) PeerInfo() protos.PeerInfo {
	peerInfo := protos.PeerInfo{
		Id:      p.host.ID().Pretty(),
		Address: p.getMultiAddr(p.host.ID(), p.host.Addrs()),
		Account: p.account,
	}

	peerStore := p.host.Peerstore()
	for _, peerID := range p.kdht.RoutingTable().ListPeers() {
		key := GenPeerIDKey(peerID)
		account, err := p.kdht.GetValue(context.Background(), key)
		if err != nil {
			p.log.Warn("get account error", "peerID", peerID)
		}

		addrInfo := peerStore.PeerInfo(peerID)
		remotePeerInfo := &protos.PeerInfo{
			Id:      peerID.String(),
			Address: p.getMultiAddr(addrInfo.ID, addrInfo.Addrs),
			Account: string(account),
		}
		peerInfo.Peer = append(peerInfo.Peer, remotePeerInfo)
	}

	return peerInfo
}

func (p *P2PServerV2) getMultiAddr(peerID peer.ID, addrs []multiaddr.Multiaddr) string {
	peerInfo := &peer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}

	multiAddrs, err := peer.AddrInfoToP2pAddrs(peerInfo)
	if err != nil {
		p.log.Warn("gen multi addr error", "peerID", p.host.ID(), "addr", p.host.Addrs())
	}

	if len(multiAddrs) >= 1 {
		return multiAddrs[0].String()
	}

	return ""
}

// ConnectPeerByAddress provide connection support using peer address(netURL)
func (p *P2PServerV2) connectPeerByAddress(addresses []string) int {
	return p.connectPeer(p.getAddrInfos(addresses))
}

func (p *P2PServerV2) getAddrInfos(addresses []string) []peer.AddrInfo {
	addrInfos := make([]peer.AddrInfo, 0, len(addresses))
	for _, addr := range addresses {
		peerAddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			p.log.Error("p2p: parse peer address error", "peerAddr", peerAddr, "error", err)
			continue
		}

		addrInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			p.log.Error("p2p: get peer node info error", "peerAddr", peerAddr, "error", err)
			continue
		}

		addrInfos = append(addrInfos, *addrInfo)
	}

	return addrInfos
}

// connectPeer connect to given peers, return the connected number of peers
// only retry if all connection failed
func (p *P2PServerV2) connectPeer(addrInfos []peer.AddrInfo) int {
	if len(addrInfos) <= 0 {
		return 0
	}

	retry := network.Retry
	success := 0
	for retry > 0 {
		for _, addrInfo := range addrInfos {
			if err := p.host.Connect(p.ctx, addrInfo); err != nil {
				p.log.Error("p2p: connection with peer node error", "error", err)
				continue
			}

			success++
			p.log.Info("p2p: connection established", "addrInfo", addrInfo)
		}

		if success > 0 {
			break
		}

		retry--
		time.Sleep(3 * time.Second)
	}

	return success
}

// SendMessage send message to peers using given filter strategy
func (p *P2PServerV2) SendMessage(ctx xctx.Context, msg *protos.CoreMessage,
	optFunc ...network.OptionFunc) error {
	ctx = &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
	tm := time.Now()
	defer func() {
		size := proto.Size(msg)
		if p.ctx.EnvCfg.MetricSwitch {
			labels := prometheus.Labels{
				metrics.LabelBCName:      msg.GetHeader().GetBcname(),
				metrics.LabelMessageType: msg.GetHeader().GetType().String(),
			}
			metrics.NetworkMsgSendCounter.With(labels).Inc()
			metrics.NetworkMsgSendBytesCounter.With(labels).Add(float64(size))
			metrics.NetworkClientHandlingHistogram.With(labels).Observe(time.Since(tm).Seconds())
		}
		ctx.GetLog().Debug("SendMessage", "log_id", msg.GetHeader().GetLogid(),
			"bcname", msg.GetHeader().GetBcname(), "msgType", msg.GetHeader().GetType(), "msgSize", size,
			"checksum", msg.GetHeader().GetDataCheckSum(), "timer", ctx.GetTimer().Print())
	}()

	opt := network.Apply(optFunc)
	filter := p.getFilter(msg, opt)
	peers, _ := filter.Filter()
	ctx.GetTimer().Mark("filter")

	var peerIDs []peer.ID
	whiteList := opt.WhiteList
	if len(whiteList) > 0 {
		for _, v := range peers {
			if _, exist := whiteList[v.Pretty()]; exist {
				peerIDs = append(peerIDs, v)
			}
		}
	} else {
		peerIDs = peers
	}

	if len(peerIDs) <= 0 {
		p.log.Warn("SendMessage peerID empty", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return ErrEmptyPeer
	}

	ctx.GetLog().SetInfoField("peerCount", len(peerIDs))
	return p.sendMessage(ctx, msg, peerIDs)
}

func (p *P2PServerV2) sendMessage(ctx xctx.Context, msg *protos.CoreMessage, peerIDs []peer.ID) error {
	var wg sync.WaitGroup
	for _, peerID := range peerIDs {
		wg.Add(1)

		go func(peerID peer.ID) {
			streamCtx := &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
			defer func() {
				wg.Done()
				streamCtx.GetLog().Debug("SendMessage", "log_id", msg.GetHeader().GetLogid(),
					"bcname", msg.GetHeader().GetBcname(), "msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(),
					"peer", peerID, "timer", streamCtx.GetTimer().Print())
			}()

			stream, err := p.streamPool.Get(ctx, peerID)
			streamCtx.GetTimer().Mark("connect")
			if err != nil {
				p.log.Warn("p2p: get stream error", "log_id", msg.GetHeader().GetLogid(),
					"msgType", msg.GetHeader().GetType(), "error", err)
				return
			}

			if err := stream.SendMessage(streamCtx, msg); err != nil {
				p.log.Error("SendMessage error", "log_id", msg.GetHeader().GetLogid(),
					"msgType", msg.GetHeader().GetType(), "error", err)
				return
			}
		}(peerID)
	}
	wg.Wait()
	ctx.GetTimer().Mark("send")
	return nil
}

// SendMessageWithResponse send message to peers using given filter strategy, expect response from peers
// 客户端再使用该方法请求带返回的消息时，最好带上log_id, 否则会导致收消息时收到不匹配的消息而影响后续的处理
func (p *P2PServerV2) SendMessageWithResponse(ctx xctx.Context, msg *protos.CoreMessage,
	optFunc ...network.OptionFunc) ([]*protos.CoreMessage, error) {
	ctx = &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
	tm := time.Now()
	defer func() {
		if p.ctx.EnvCfg.MetricSwitch {
			labels := prometheus.Labels{
				metrics.LabelBCName:      msg.GetHeader().GetBcname(),
				metrics.LabelMessageType: msg.GetHeader().GetType().String(),
			}
			metrics.NetworkMsgSendCounter.With(labels).Inc()
			metrics.NetworkMsgSendBytesCounter.With(labels).Add(float64(proto.Size(msg)))
			metrics.NetworkClientHandlingHistogram.With(labels).Observe(time.Since(tm).Seconds())
		}
		ctx.GetLog().Debug("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
			"bcname", msg.GetHeader().GetBcname(), "msgType", msg.GetHeader().GetType(),
			"checksum", msg.GetHeader().GetDataCheckSum(), "timer", ctx.GetTimer().Print())
	}()

	opt := network.Apply(optFunc)
	filter := p.getFilter(msg, opt)
	peers, _ := filter.Filter()

	var peerIDs []peer.ID
	// 做一层过滤(基于白名单过滤)
	whiteList := opt.WhiteList
	if len(whiteList) > 0 {
		for _, v := range peers {
			if _, exist := whiteList[v.Pretty()]; exist {
				peerIDs = append(peerIDs, v)
			}
		}
	} else {
		peerIDs = peers
	}
	ctx.GetTimer().Mark("filter")

	if len(peerIDs) <= 0 {
		p.log.Warn("SendMessageWithResponse peerID empty", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return nil, ErrEmptyPeer
	}

	ctx.GetLog().SetInfoField("peerCount", len(peerIDs))
	return p.sendMessageWithResponse(ctx, msg, peerIDs, opt)
}

func (p *P2PServerV2) sendMessageWithResponse(ctx xctx.Context, msg *protos.CoreMessage,
	peerIDs []peer.ID, opt *network.Option) ([]*protos.CoreMessage, error) {

	respCh := make(chan *protos.CoreMessage, len(peerIDs))
	var wg sync.WaitGroup
	ctx.GetLog().Debug("sendMessageWithResponse peers", "peers", peerIDs)
	for _, peerID := range peerIDs {
		wg.Add(1)
		go func(peerID peer.ID) {
			streamCtx := &xctx.BaseCtx{XLog: ctx.GetLog(), Timer: timer.NewXTimer()}
			defer func() {
				wg.Done()
				streamCtx.GetLog().Debug("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
					"bcname", msg.GetHeader().GetBcname(), "msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(),
					"peer", peerID, "timer", streamCtx.GetTimer().Print())
			}()

			stream, err := p.streamPool.Get(ctx, peerID)
			streamCtx.GetTimer().Mark("connect")
			if err != nil {
				p.log.Warn("p2p: get stream error", "log_id", msg.GetHeader().GetLogid(),
					"msgType", msg.GetHeader().GetType(), "error", err)
				return
			}

			resp, err := stream.SendMessageWithResponse(streamCtx, msg)
			if err != nil {
				p.log.Warn("p2p: SendMessageWithResponse error", "log_id", msg.GetHeader().GetLogid(),
					"msgType", msg.GetHeader().GetType(), "error", err)
				return
			}

			respCh <- resp
		}(peerID)
	}
	wg.Wait()
	ctx.GetTimer().Mark("send")

	if len(respCh) <= 0 {
		p.log.Warn("p2p: no response", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType())
		return nil, ErrNoResponse
	}

	i := 0
	length := len(respCh)
	threshold := int(float32(len(peerIDs)) * opt.Percent)
	response := make([]*protos.CoreMessage, 0, len(peerIDs))
	for resp := range respCh {
		if network.VerifyChecksum(resp) {
			response = append(response, resp)
		}

		i++
		if i >= length || len(response) >= threshold {
			break
		}
	}

	ctx.GetTimer().Mark("recv")
	return response, nil
}

func (p *P2PServerV2) getFilter(msg *protos.CoreMessage, opt *network.Option) PeerFilter {
	if len(opt.Filters) <= 0 && len(opt.Addresses) <= 0 &&
		len(opt.PeerIDs) <= 0 && len(opt.Accounts) <= 0 {
		opt.Filters = []network.FilterStrategy{network.DefaultStrategy}
	}

	bcname := msg.GetHeader().GetBcname()
	if len(p.getStaticNodes(bcname)) != 0 {
		return &StaticNodeStrategy{srv: p, bcname: bcname}
	}

	peerFilters := make([]PeerFilter, 0)
	for _, strategy := range opt.Filters {
		var filter PeerFilter
		switch strategy {
		case network.NearestBucketStrategy:
			filter = &NearestBucketFilter{srv: p}
		case network.BucketsWithFactorStrategy:
			filter = &BucketsFilterWithFactor{srv: p, factor: opt.Factor}
		default:
			filter = &BucketsFilter{srv: p}
		}
		peerFilters = append(peerFilters, filter)
	}

	peerIDs := make([]peer.ID, 0)
	if len(opt.Addresses) > 0 {
		go p.connectPeerByAddress(opt.Addresses)
		for _, addr := range opt.Addresses {
			peerID, err := GetPeerIDByAddress(addr)
			if err != nil {
				p.log.Warn("p2p: getFilter parse peer address failed", "paddr", addr, "error", err)
				continue
			}
			peerIDs = append(peerIDs, peerID)
		}
	}

	if len(opt.Accounts) > 0 {
		for _, account := range opt.Accounts {
			peerID, err := p.GetPeerIdByAccount(account)
			if err != nil {
				p.log.Warn("p2p: getFilter get peer id by account failed", "account", account, "error", err)
				continue
			}
			peerIDs = append(peerIDs, peerID)
		}
	}

	if len(opt.PeerIDs) > 0 {
		for _, encodedPeerID := range opt.PeerIDs {
			peerID, err := peer.Decode(encodedPeerID)
			if err != nil {
				p.log.Warn("p2p: getFilter parse peer ID failed", "pid", peerID, "error", err)
				continue
			}
			peerIDs = append(peerIDs, peerID)
		}
	}
	return NewMultiStrategy(peerFilters, peerIDs)
}

func (p *P2PServerV2) GetPeerIdByAccount(account string) (peer.ID, error) {
	if value, ok := p.accounts.Get(account); ok {
		return value.(peer.ID), nil
	}

	key := Key(account)
	value, err := p.kdht.GetValue(context.Background(), key)
	if err != nil {
		return "", fmt.Errorf("dht get peer id error: %s", err)
	}

	peerID, err := GetPeerIDByAddress(string(value))
	if err != nil {
		return "", fmt.Errorf("address error: %s, address=%s", err, value)
	}

	p.accounts.Set(key, peerID, cache.NoExpiration)
	return peerID, nil
}

// GetStaticNodes get StaticNode a chain
func (p *P2PServerV2) getStaticNodes(bcname string) []peer.ID {
	return p.staticNodes[bcname]
}
