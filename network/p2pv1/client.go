package p2pv1

import (
	"errors"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/metrics"
	"github.com/wooyang2018/corechain/network"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

var (
	ErrEmptyPeer  = errors.New("empty peer")
	ErrNoResponse = errors.New("no response")
)

// SendMessage send message to peers using given filter strategy
func (p *P2PServerV1) SendMessage(ctx xctx.Context, msg *protos.CoreMessage, optFunc ...network.OptionFunc) error {
	if p.ctx.EnvCfg.MetricSwitch {
		tm := time.Now()
		defer func() {
			labels := prometheus.Labels{
				metrics.LabelBCName:      msg.GetHeader().GetBcname(),
				metrics.LabelMessageType: msg.GetHeader().GetType().String(),
			}
			metrics.NetworkMsgSendCounter.With(labels).Inc()
			metrics.NetworkMsgSendBytesCounter.With(labels).Add(float64(proto.Size(msg)))
			metrics.NetworkClientHandlingHistogram.With(labels).Observe(time.Since(tm).Seconds())
		}()
	}

	opt := network.Apply(optFunc) //根据optFunc构造option
	filter := p.getFilter(msg, opt)
	peerIDs, err := filter.Filter()
	if err != nil {
		p.log.Warn("network: filter error", "log_id", msg.GetHeader().GetLogid())
		return errors.New("network SendMessage: filter returned error data")
	}

	if len(peerIDs) <= 0 {
		p.log.Warn("SendMessage peerID empty", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return ErrEmptyPeer
	}

	p.log.Debug("SendMessage", "log_id", msg.GetHeader().GetLogid(),
		"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "peerID", peerIDs)
	return p.sendMessage(ctx, msg, peerIDs)
}

func (p *P2PServerV1) sendMessage(ctx xctx.Context, msg *protos.CoreMessage, peerIDs []string) error {
	wg := sync.WaitGroup{}
	for _, peerID := range peerIDs {
		peerID := peerID
		conn, err := p.pool.Get(peerID)
		if err != nil {
			p.log.Warn("network: get conn error",
				"log_id", msg.GetHeader().GetLogid(), "peerID", peerID, "error", err)
			continue
		}

		wg.Add(1)
		go func(conn *Conn) {
			defer wg.Done()
			err = conn.SendMessage(ctx, msg)
			if err != nil {
				p.log.Warn("network: SendMessage error",
					"log_id", msg.GetHeader().GetLogid(), "peerID", conn.id, "error", err)
			}
		}(conn)
	}
	wg.Wait()

	return nil
}

// SendMessageWithResponse send message to peers using given filter strategy, expect response from peers
func (p *P2PServerV1) SendMessageWithResponse(ctx xctx.Context, msg *protos.CoreMessage, optFunc ...network.OptionFunc) ([]*protos.CoreMessage, error) {
	if p.ctx.EnvCfg.MetricSwitch {
		tm := time.Now()
		defer func() {
			labels := prometheus.Labels{
				metrics.LabelBCName:      msg.GetHeader().GetBcname(),
				metrics.LabelMessageType: msg.GetHeader().GetType().String(),
			}
			metrics.NetworkMsgSendCounter.With(labels).Inc()
			metrics.NetworkMsgSendBytesCounter.With(labels).Add(float64(proto.Size(msg)))
			metrics.NetworkClientHandlingHistogram.With(labels).Observe(time.Since(tm).Seconds())
		}()
	}

	opt := network.Apply(optFunc)
	filter := p.getFilter(msg, opt)
	peerIDs, err := filter.Filter()
	if err != nil {
		p.log.Warn("network: filter error", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return nil, errors.New("network: SendMessageWithRes: filter returned error data")
	}

	if len(peerIDs) <= 0 {
		p.log.Warn("SendMessageWithResponse peerID empty", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum())
		return nil, ErrEmptyPeer
	}

	p.log.Debug("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
		"msgType", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "peerID", peerIDs)
	return p.sendMessageWithResponse(ctx, msg, peerIDs, opt.Percent)
}

func (p *P2PServerV1) sendMessageWithResponse(ctx xctx.Context, msg *protos.CoreMessage, peerIDs []string, percent float32) ([]*protos.CoreMessage, error) {
	wg := sync.WaitGroup{}
	respCh := make(chan *protos.CoreMessage, len(peerIDs))
	for _, peerID := range peerIDs {
		peerID := peerID
		conn, err := p.pool.Get(peerID)
		if err != nil {
			p.log.Warn("network: get conn error", "log_id", msg.GetHeader().GetLogid(),
				"peerID", peerID, "error", err)
			continue
		}

		wg.Add(1)
		go func(conn *Conn) {
			defer wg.Done()

			resp, err := conn.SendMessageWithResponse(ctx, msg)
			if err != nil {
				return
			}
			resp.Header.From = peerID
			respCh <- resp
		}(conn)
	}
	wg.Wait()

	if len(respCh) <= 0 {
		p.log.Warn("network: no response", "log_id", msg.GetHeader().GetLogid())
		return nil, ErrNoResponse
	}

	i := 0
	length := len(respCh)
	threshold := int(float32(len(peerIDs)) * percent)
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

	return response, nil
}

func (p *P2PServerV1) getFilter(msg *protos.CoreMessage, opt *network.Option) PeerFilter {
	if len(opt.Filters) <= 0 && len(opt.Addresses) <= 0 &&
		len(opt.PeerIDs) <= 0 && len(opt.Accounts) <= 0 {
		opt.Filters = []network.FilterStrategy{network.DefaultStrategy}
	}

	bcname := msg.GetHeader().GetBcname()
	peerFilters := make([]PeerFilter, 0)
	for _, f := range opt.Filters {
		var filter PeerFilter
		switch f {
		default:
			filter = &StaticNodeStrategy{broadcast: p.config.IsBroadCast, srv: p, bcname: bcname}
		}
		peerFilters = append(peerFilters, filter)
	}

	peerIDs := make([]string, 0)
	if len(opt.PeerIDs) > 0 {
		peerIDs = append(peerIDs, opt.PeerIDs...)
	}

	if len(opt.Addresses) > 0 {
		peerIDs = append(peerIDs, opt.Addresses...)
	}

	if len(opt.Accounts) > 0 {
		for _, account := range opt.Accounts {
			peerID, err := p.GetPeerIdByAccount(account)
			if err != nil {
				p.log.Warn("network: getFilter get peer id by account failed", "account", account, "error", err)
				continue
			}
			peerIDs = append(peerIDs, peerID)
		}
	}

	return NewMultiStrategy(peerFilters, peerIDs)
}

func (p *P2PServerV1) GetPeerIdByAccount(account string) (string, error) {
	if value, ok := p.accounts.Get(account); ok {
		return value.(string), nil
	}
	if !p.pool.staticModeOn {
		return "", ErrAccountNotExist
	}
	// xchain address can not mapping, try getPeerInfo again.
	addresses := make(map[string]struct{})
	for _, nodes := range p.staticNodes {
		for _, node := range nodes {
			if _, ok := addresses[node]; ok {
				continue
			}
			addresses[node] = struct{}{}
		}
	}
	am := p.accounts.Items()
	for _, v := range am {
		addr, _ := v.Object.(string)
		delete(addresses, addr)
	}
	var retryPeers []string
	for k, _ := range addresses {
		retryPeers = append(retryPeers, k)
	}
	p.GetPeerInfo(retryPeers)
	// retry
	if value, ok := p.accounts.Get(account); ok {
		return value.(string), nil
	}
	return "", ErrAccountNotExist
}
