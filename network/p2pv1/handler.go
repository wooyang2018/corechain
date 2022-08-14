package p2pv1

import (
	"errors"
	"sync"

	"github.com/patrickmn/go-cache"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/network"
	"github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/protos"
)

func (p *P2PServerV1) GetPeerInfo(addresses []string) ([]*protos.PeerInfo, error) {
	if len(addresses) == 0 {
		return nil, errors.New("neighbors empty")
	}
	peerInfo := p.PeerInfo()
	p.accounts.Set(peerInfo.GetAccount(), peerInfo.GetAddress(), 0)

	var remotePeers []*protos.PeerInfo
	var wg sync.WaitGroup
	var mutex sync.Mutex
	for _, addr := range addresses {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			rps := p.GetPeer(peerInfo, addr)
			if rps == nil {
				return
			}
			mutex.Lock()
			remotePeers = append(remotePeers, rps...)
			mutex.Unlock()
		}(addr)
	}
	wg.Wait()
	return remotePeers, nil
}

func (p *P2PServerV1) GetPeer(peerInfo protos.PeerInfo, addr string) []*protos.PeerInfo {
	var remotePeers []*protos.PeerInfo
	msg := network.NewMessage(protos.CoreMessage_GET_PEER_INFO, &peerInfo)
	response, err := p.SendMessageWithResponse(p.ctx, msg, base.WithAddresses([]string{addr}))
	if err != nil {
		p.log.Error("get peer error", "log_id", msg.GetHeader().GetLogid(), "error", err)
		return nil
	}
	for _, msg := range response {
		var peer protos.PeerInfo
		err := network.Unmarshal(msg, &peer)
		if err != nil {
			p.log.Warn("unmarshal NewNode response error", "log_id", msg.GetHeader().GetLogid(), "error", err)
			continue
		}
		peer.Address = addr
		p.accounts.Set(peer.GetAccount(), peer.GetAddress(), cache.NoExpiration)
		remotePeers = append(remotePeers, &peer)
	}
	return remotePeers
}

func (p *P2PServerV1) registerConnectHandler() error {
	err := p.Register(network.NewSubscriber(p.ctx, protos.CoreMessage_GET_PEER_INFO, network.HandleFunc(p.handleGetPeerInfo)))
	if err != nil {
		p.log.Error("registerSubscribe error", "error", err)
		return err
	}

	return nil
}

func (p *P2PServerV1) handleGetPeerInfo(ctx xctx.Context, request *protos.CoreMessage) (*protos.CoreMessage, error) {
	output := p.PeerInfo()
	opts := []network.MessageOption{
		network.WithBCName(request.GetHeader().GetBcname()),
		network.WithErrorType(protos.CoreMessage_SUCCESS),
		network.WithLogId(request.GetHeader().GetLogid()),
	}
	resp := network.NewMessage(protos.CoreMessage_GET_PEER_INFO_RES, &output, opts...)

	var peerInfo protos.PeerInfo
	err := network.Unmarshal(request, &peerInfo)
	if err != nil {
		p.log.Warn("unmarshal NewNode response error", "error", err)
		return resp, nil
	}

	if !p.pool.staticModeOn {
		uniq := make(map[string]struct{}, len(p.dynamicNodes))
		for _, address := range p.dynamicNodes {
			uniq[address] = struct{}{}
		}

		if _, ok := uniq[peerInfo.Address]; !ok {
			p.dynamicNodes = append(p.dynamicNodes, peerInfo.Address)
		}
		p.accounts.Set(peerInfo.GetAccount(), peerInfo.GetAddress(), cache.NoExpiration)
	}

	return resp, nil
}
