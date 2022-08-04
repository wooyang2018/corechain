package p2pv2

import (
	"errors"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
	libnet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/wooyang2018/corechain/common/cache"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/network"
	nctx "github.com/wooyang2018/corechain/network/context"
)

// define base errors
var (
	ErrStreamPoolFull = errors.New("stream pool is full")
)

// StreamPool manage all the stream
type StreamPool struct {
	dispatcher     network.Dispatcher
	ctx            *nctx.NetCtx
	log            logger.Logger
	host           host.Host
	kdht           *dht.IpfsDHT
	limit          *StreamLimit
	mutex          sync.Mutex
	streams        *cache.LRUCache // key: peer id, value: Stream
	maxStreamLimit int32
}

// NewStreamPool create StreamPool instance
func NewStreamPool(ctx *nctx.NetCtx, ho host.Host, dispatcher network.Dispatcher) (*StreamPool, error) {
	cfg := ctx.P2PConf
	limit := &StreamLimit{}
	limit.Init(ctx)
	return &StreamPool{
		ctx:            ctx,
		log:            ctx.GetLog(),
		limit:          limit,
		host:           ho,
		dispatcher:     dispatcher,
		mutex:          sync.Mutex{},
		streams:        cache.NewLRUCache(int(cfg.MaxStreamLimits)),
		maxStreamLimit: cfg.MaxStreamLimits,
	}, nil
}

// Get will probe and return a stream
func (sp *StreamPool) Get(ctx xctx.Context, peerId peer.ID) (*StreamImpl, error) {

	if v, ok := sp.streams.Get(peerId.Pretty()); ok {
		if stream, ok := v.(*StreamImpl); ok {
			if stream.Valid() {
				return stream, nil
			} else {
				sp.DelStream(stream)
				ctx.GetLog().Warn("stream not valid, create new stream", "peerId", peerId)
			}
		}
	}

	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	if v, ok := sp.streams.Get(peerId.Pretty()); ok {
		if stream, ok := v.(*StreamImpl); ok {
			if stream.Valid() {
				return stream, nil
			} else {
				sp.DelStream(stream)
				ctx.GetLog().Warn("stream not valid, create new stream", "peerId", peerId)
			}
		}
	}

	netStream, err := sp.host.NewStream(sp.ctx, peerId, network.ProtocolVersion)
	if err != nil {
		if errors.Is(err, swarm.ErrDialToSelf) {
			ctx.GetLog().Info("new net stream error", "peerId", peerId, "error", err)
			return nil, ErrNewStream
		}
		ctx.GetLog().Warn("new net stream error", "peerId", peerId, "error", err)
		return nil, ErrNewStream
	}

	return sp.NewStream(ctx, netStream)
}

// Add used to add a new net stream into pool
func (sp *StreamPool) NewStream(ctx xctx.Context, netStream libnet.Stream) (*StreamImpl, error) {
	stream, err := NewStream(sp.ctx, netStream, sp.dispatcher, sp.host)
	if err != nil {
		return nil, err
	}

	if err := sp.AddStream(ctx, stream); err != nil {
		stream.Close()
		sp.kdht.RoutingTable().RemovePeer(stream.id)
		ctx.GetLog().Warn("New stream is deleted", "error", err)
		return nil, ErrNewStream
	}

	return stream, nil
}

// AddStream used to add a new P2P stream into pool
func (sp *StreamPool) AddStream(ctx xctx.Context, stream *StreamImpl) error {
	multiAddr := stream.MultiAddr()
	ok := sp.limit.AddStream(multiAddr.String(), stream.id)
	if !ok || int32(sp.streams.Len()) > sp.maxStreamLimit {
		ctx.GetLog().Warn("add stream limit error", "peerID", stream.id, "multiAddr", multiAddr, "error", "over limit")
		return ErrStreamPoolFull
	}

	if v, ok := sp.streams.Get(stream.id.Pretty()); ok {
		ctx.GetLog().Warn("replace stream", "peerID", stream.id, "multiAddr", multiAddr)
		if s, ok := v.(*StreamImpl); ok {
			sp.DelStream(s)
		}
	}

	sp.streams.Add(stream.id.Pretty(), stream)
	return nil
}

// DelStream delete a stream
func (sp *StreamPool) DelStream(stream *StreamImpl) error {
	stream.Close()
	sp.streams.Del(stream.PeerID())
	sp.limit.DelStream(stream.MultiAddr().String())
	return nil
}
