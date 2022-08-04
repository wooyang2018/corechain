package p2pv1

import (
	"sync"

	nctx "github.com/wooyang2018/corechain/network/context"
	"google.golang.org/grpc/connectivity"
)

// ConnPool manage all the connection
type ConnPool struct {
	ctx  *nctx.NetCtx
	pool sync.Map // map[peerID]*conn

	staticModeOn bool
}

func NewConnPool(ctx *nctx.NetCtx) (*ConnPool, error) {
	cp := ConnPool{
		ctx: ctx,
	}
	if len(ctx.P2PConf.BootNodes) == 0 && len(ctx.P2PConf.StaticNodes) > 0 {
		cp.staticModeOn = true
	}
	return &cp, nil
}

func (p *ConnPool) Get(addr string) (*Conn, error) {
	if v, ok := p.pool.Load(addr); ok {
		return v.(*Conn), nil
	}

	conn, err := NewConn(p.ctx, addr)
	if err != nil {
		return nil, err
	}

	p.pool.LoadOrStore(addr, conn)
	return conn, nil
}

func (p *ConnPool) GetAll() map[string]string {
	remotePeer := make(map[string]string, 32)
	p.pool.Range(func(key, value interface{}) bool {
		addr := key.(string)
		conn := value.(*Conn)
		if conn.conn.GetState() == connectivity.Ready {
			remotePeer[conn.PeerID()] = addr
		}
		return true
	})

	return remotePeer
}
