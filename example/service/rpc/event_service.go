package rpc

import (
	"errors"
	"net"
	"sync"

	protos2 "github.com/wooyang2018/corechain/example/protos"
	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"

	engineBase "github.com/wooyang2018/corechain/engines/base"
	"github.com/wooyang2018/corechain/engines/event"
	scom "github.com/wooyang2018/corechain/example/service/common"
	sconf "github.com/wooyang2018/corechain/example/service/config"
)

// eventService implements the interface of pb.EventService
type eventService struct {
	cfg    *sconf.ServConf
	router *event.Router

	mutex       sync.Mutex
	connCounter map[string]int
}

func newEventService(cfg *sconf.ServConf, engine engineBase.Engine) *eventService {
	return &eventService{
		cfg:         cfg,
		router:      event.NewRouter(engine),
		connCounter: make(map[string]int),
	}
}

// Subscribe start an event subscribe
func (e *eventService) Subscribe(req *protos2.SubscribeRequest, stream protos2.EventService_SubscribeServer) error {
	if !e.cfg.EnableEvent {
		return errors.New("event service disabled")
	}

	// check same ip limit
	remoteIP, err := e.connPermit(stream.Context())
	if err != nil {
		return err
	}
	defer e.releaseConn(remoteIP)

	encfunc, iter, err := e.router.Subscribe(scom.ConvertEventSubType(req.GetType()), req.GetFilter())
	if err != nil {
		return err
	}
	for iter.Next() {
		payload := iter.Data()
		buf, _ := encfunc(payload)
		event := &protos2.Event{
			Payload: buf,
		}
		err := stream.Send(event)
		if err != nil {
			break
		}
	}
	iter.Close()

	if iter.Error() != nil {
		return iter.Error()
	}
	return nil
}

func (e *eventService) connPermit(ctx context.Context) (string, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return "", errors.New("get remote address error")
	}
	remoteIP, _, err := net.SplitHostPort(peer.Addr.String())
	if err != nil {
		return "", err
	}

	if e.cfg.EventAddrMaxConn == 0 {
		return remoteIP, nil
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()
	cnt, ok := e.connCounter[remoteIP]
	if !ok {
		e.connCounter[remoteIP] = 1
		return remoteIP, nil
	}
	if cnt >= e.cfg.EventAddrMaxConn {
		return "", errors.New("maximum connections exceeded")
	}
	e.connCounter[remoteIP]++
	return remoteIP, nil
}

func (e *eventService) releaseConn(addr string) {
	if e.cfg.EventAddrMaxConn == 0 {
		return
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.connCounter[addr] <= 1 {
		delete(e.connCounter, addr)
		return
	}
	e.connCounter[addr]--
}