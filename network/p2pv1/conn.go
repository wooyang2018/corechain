package p2pv1

import (
	"context"
	"errors"
	"io"
	"time"

	xctx "github.com/wooyang2018/corechain/common/context"
	"github.com/wooyang2018/corechain/logger"
	"github.com/wooyang2018/corechain/network"
	netBase "github.com/wooyang2018/corechain/network/base"
	"github.com/wooyang2018/corechain/network/config"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

type Conn struct {
	ctx    *netBase.NetCtx
	log    logger.Logger
	config *config.NetConf

	id   string // addr:"IP:Port"
	conn *grpc.ClientConn
}

// NewConn create new connection with addr
func NewConn(ctx *netBase.NetCtx, addr string) (*Conn, error) {
	c := &Conn{
		id:     addr,
		config: ctx.P2PConf,
		log:    ctx.GetLog(),
	}

	if err := c.newConn(); err != nil {
		ctx.GetLog().Error("NewConn error", "error", err)
		return nil, err
	}

	return c, nil
}

func (c *Conn) newClient() (protos.P2PServiceClient, error) {
	state := c.conn.GetState()
	if state == connectivity.TransientFailure || state == connectivity.Shutdown {
		c.log.Error("newClient conn state not ready", "id", c.id, "state", state.String())
		c.Close()
		err := c.newConn()
		if err != nil {
			c.log.Error("newClient newGrpcConn error", "id", c.id, "error", err)
			return nil, err
		}
	}

	return protos.NewP2PServiceClient(c.conn), nil
}

func (c *Conn) newConn() error {
	options := append([]grpc.DialOption{}, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(c.config.MaxMessageSize)<<20)))
	if c.config.IsTls {
		creds, err := network.NewTLS(c.config.KeyPath, c.config.ServiceName)
		if err != nil {
			return err
		}
		options = append(options, grpc.WithTransportCredentials(creds))
	} else {
		options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(c.id, options...)
	if err != nil {
		c.log.Error("newGrpcConn error", "error", err, "peerID", c.id)
		return errors.New("new grpc conn error")
	}

	c.conn = conn
	return nil
}

// SendMessage send message to a peer
func (c *Conn) SendMessage(ctx xctx.Context, msg *protos.CoreMessage) error {
	client, err := c.newClient()
	if err != nil {
		c.log.Error("SendMessage new client error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return err
	}

	sctx, cancel := context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
	defer cancel()
	stream, err := client.SendP2PMessage(sctx)
	if err != nil {
		c.log.Error("SendMessage new stream error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return err
	}
	defer stream.CloseSend()

	c.log.Debug("SendMessage", "log_id", msg.GetHeader().GetLogid(),
		"type", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "peerID", c.id)

	msg.Header.From = c.config.Address
	err = stream.Send(msg)
	if err != nil {
		c.log.Error("SendMessage Send error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return err
	}
	if err == io.EOF {
		return nil
	}

	// client等待server收到消息再退出，防止提前退出导致信息发送失败
	_, err = stream.Recv()

	return err
}

// SendMessageWithResponse send message to a peer with responce
func (c *Conn) SendMessageWithResponse(ctx xctx.Context, msg *protos.CoreMessage) (*protos.CoreMessage, error) {
	client, err := c.newClient() //新建到远端节点的连接
	if err != nil {
		c.log.Error("SendMessageWithResponse new client error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return nil, err
	}

	sctx, cancel := context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
	defer cancel()
	stream, err := client.SendP2PMessage(sctx)
	if err != nil {
		c.log.Error("SendMessageWithResponse new stream error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return nil, err
	}
	defer stream.CloseSend()

	c.log.Debug("SendMessageWithResponse", "log_id", msg.GetHeader().GetLogid(),
		"type", msg.GetHeader().GetType(), "checksum", msg.GetHeader().GetDataCheckSum(), "peerID", c.id)

	msg.Header.From = c.config.Address
	err = stream.Send(msg)
	if err != nil {
		c.log.Error("SendMessageWithResponse error", "log_id", msg.GetHeader().GetLogid(), "error", err, "peerID", c.id)
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		c.log.Error("SendMessageWithResponse Recv error", "log_id", resp.GetHeader().GetLogid(), "error", err.Error())
		return nil, err
	}

	c.log.Debug("SendMessageWithResponse return", "log_id", resp.GetHeader().GetLogid(), "peerID", c.id)
	return resp, nil
}

// Close close this conn
func (c *Conn) Close() {
	c.log.Info("Conn Close", "peerID", c.id)
	c.conn.Close()
}

// PeerID return conn id
func (c *Conn) PeerID() string {
	return c.id
}
