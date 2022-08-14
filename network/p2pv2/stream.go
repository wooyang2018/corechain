package p2pv2

import (
	"bufio"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	libnet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/common/metrics"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/network"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/network/config"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

// define base errors
var (
	ErrNewStream       = errors.New("new stream error")
	ErrStreamNotValid  = errors.New("stream not valid")
	ErrNoneMessageType = errors.New("none message type")
)

// Stream is the IO wrapper for underly P2P connection
type StreamImpl struct {
	ctx        *netBase.NetCtx
	config     *config.NetConf
	log        logger.Logger
	dispatcher netBase.Dispatcher
	stream     libnet.Stream
	streamMu   *sync.Mutex
	id         peer.ID
	host       host.Host
	addr       multiaddr.Multiaddr
	w          *bufio.Writer
	wc         WriteCloser
	rc         ReadCloser
	valid      bool
	grpcPort   string
}

// NewStream create Stream instance
func NewStream(ctx *netBase.NetCtx, netStream libnet.Stream, dispatcher netBase.Dispatcher, host host.Host) (*StreamImpl, error) {
	w := bufio.NewWriter(netStream)
	wc := NewDelimitedWriter(w)
	maxMsgSize := int(ctx.P2PConf.MaxMessageSize) << 20
	stream := &StreamImpl{
		ctx:        ctx,
		config:     ctx.P2PConf,
		log:        ctx.GetLog(),
		dispatcher: dispatcher,
		stream:     netStream,
		host:       host,
		streamMu:   new(sync.Mutex),
		id:         netStream.Conn().RemotePeer(),
		addr:       netStream.Conn().RemoteMultiaddr(),
		rc:         NewDelimitedReader(netStream, maxMsgSize),
		w:          w,
		wc:         wc,
		valid:      true,
	}
	stream.Start()
	return stream, nil
}

// PeerID get id
func (s *StreamImpl) PeerID() string {
	return s.id.Pretty()
}

// MultiAddr get multi addr
func (s *StreamImpl) MultiAddr() multiaddr.Multiaddr {
	return s.addr
}

// Start used to start
func (s *StreamImpl) Start() {
	go s.LoopRecv()
}

// Close close the connected IO stream
func (s *StreamImpl) Close() {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	s.resetLockFree()
}

func (s *StreamImpl) resetLockFree() {
	if s.Valid() {
		if s.stream != nil {
			s.stream.Reset()
		}
		s.stream = nil
		s.valid = false
	}
}

func (s *StreamImpl) Valid() bool {
	return s.valid
}

func (s *StreamImpl) Send(msg *protos.CoreMessage) error {
	if !s.Valid() {
		return ErrStreamNotValid
	}
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	if !s.Valid() {
		return ErrStreamNotValid
	}

	deadline := time.Now().Add(time.Duration(s.config.Timeout) * time.Second)
	s.stream.SetWriteDeadline(deadline)
	msg.Header.From = s.host.ID().Pretty()
	if err := s.wc.WriteMsg(msg); err != nil {
		s.resetLockFree()
		return err
	}
	return s.w.Flush()
}

// LoopRecv loop to read data from stream
func (s *StreamImpl) LoopRecv() {
	for {
		msg := new(protos.CoreMessage)
		err := s.rc.ReadMsg(msg)
		switch err {
		case io.EOF:
			s.log.Debug("Stream LoopRecv", "error", "io.EOF")
			s.Close()
			return
		case nil:
		default:
			s.log.Debug("Stream LoopRecv error to reset", "error", err)
			s.Close()
			return
		}
		err = s.HandleMessage(msg)
		if err != nil {
			s.Close()
			return
		}
		msg = nil
	}
}

// SendMessage will send a message to a peer
func (s *StreamImpl) SendMessage(ctx xctx.Context, msg *protos.CoreMessage) error {
	err := s.Send(msg)
	ctx.GetTimer().Mark("write")
	if err != nil {
		s.log.Error("Stream SendMessage error", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "error", err)
		return err
	}

	return nil
}

// SendMessageWithResponse will send a message to a peer and wait for response
func (s *StreamImpl) SendMessageWithResponse(ctx xctx.Context,
	msg *protos.CoreMessage) (*protos.CoreMessage, error) {
	respType := network.GetRespMessageType(msg.GetHeader().GetType())
	if respType == protos.CoreMessage_MSG_TYPE_NONE {
		return nil, ErrNoneMessageType
	}

	observerCh := make(chan *protos.CoreMessage, 100)
	sub := network.NewSubscriber(s.ctx, respType, observerCh, network.WithFilterFrom(s.PeerID()))
	err := s.dispatcher.Register(sub)
	if err != nil {
		s.log.Error("sendMessageWithResponse register error", "error", err)
		return nil, err
	}
	defer s.dispatcher.UnRegister(sub)

	errCh := make(chan error, 1)
	respCh := make(chan *protos.CoreMessage, 1)
	go func() {
		resp, err := s.waitResponse(ctx, msg, observerCh)
		if resp != nil {
			respCh <- resp
		}
		if err != nil {
			errCh <- err
		}
	}()

	// 开始写消息
	err = s.Send(msg)
	ctx.GetTimer().Mark("write")
	if err != nil {
		s.log.Warn("SendMessageWithResponse Send error", "log_id", msg.GetHeader().GetLogid(),
			"msgType", msg.GetHeader().GetType(), "err", err)
		return nil, err
	}

	// 等待返回
	select {
	case resp := <-respCh:
		return resp, nil
	case err := <-errCh:
		return nil, err
	}
}

// waitResponse wait resp with timeout
func (s *StreamImpl) waitResponse(ctx xctx.Context, msg *protos.CoreMessage,
	observerCh chan *protos.CoreMessage) (*protos.CoreMessage, error) {

	timeout := s.config.Timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			s.log.Warn("waitResponse ctx done", "log_id", msg.GetHeader().GetLogid(),
				"type", msg.GetHeader().GetType(), "pid", s.id.Pretty(), "error", timeoutCtx.Err())
			ctx.GetTimer().Mark("wait")
			return nil, timeoutCtx.Err()
		case resp := <-observerCh:
			if network.VerifyMessageType(msg, resp, s.id.Pretty()) {
				ctx.GetTimer().Mark("read")
				return resp, nil
			}

			s.log.Debug("waitResponse get resp continue", "log_id", resp.GetHeader().GetLogid(),
				"type", resp.GetHeader().GetType(), "checksum", resp.GetHeader().GetDataCheckSum(),
				"resp.from", resp.GetHeader().GetFrom())
			continue
		}
	}
}

func (s *StreamImpl) HandleMessage(msg *protos.CoreMessage) error {
	if s.dispatcher == nil {
		s.log.Warn("DispatcherImpl not ready, omit", "msg", msg)
		return nil
	}

	if s.ctx.EnvCfg.MetricSwitch {
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

	if err := s.dispatcher.Dispatch(msg, s); err != nil {
		s.log.Warn("handle new message dispatch error", "log_id", msg.GetHeader().GetLogid(),
			"type", msg.GetHeader().GetType(), "from", msg.GetHeader().GetFrom(), "error", err)
		return nil // not return err
	}

	return nil
}
