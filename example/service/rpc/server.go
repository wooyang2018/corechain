package rpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"reflect"
	"runtime"
	"strings"

	sctx "github.com/wooyang2018/corechain/example/base"
	"github.com/wooyang2018/corechain/example/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/wooyang2018/corechain/common/utils"
	"github.com/wooyang2018/corechain/engine/base"
	"github.com/wooyang2018/corechain/logger"
)

type RpcServer struct {
	engine base.Engine
	log    logger.Logger
}

func NewRpcServ(engine base.Engine, log logger.Logger) *RpcServer {
	return &RpcServer{
		engine: engine,
		log:    log,
	}
}

// UnaryInterceptor provides a hook to intercept the execution of a unary RPC on the server.
func (t *RpcServer) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (respRes interface{}, err error) {
		// set request header
		type HeaderInterface interface {
			GetHeader() *pb.Header
		}
		if req.(HeaderInterface).GetHeader() == nil {
			header := reflect.ValueOf(req).Elem().FieldByName("Header")
			if header.IsValid() && header.IsNil() && header.CanSet() {
				header.Set(reflect.ValueOf(t.defReqHeader()))
			}
		}
		if req.(HeaderInterface).GetHeader().GetLogid() == "" {
			req.(HeaderInterface).GetHeader().Logid = utils.GenLogId()
		}
		reqHeader := req.(HeaderInterface).GetHeader()

		// set request context
		reqCtx, _ := t.createReqCtx(ctx, reqHeader)
		ctx = sctx.WithReqCtx(ctx, reqCtx)

		// output access log
		logFields := make([]interface{}, 0)
		logFields = append(logFields, "from", reqHeader.GetFromNode(),
			"client_ip", reqCtx.GetClientIp(), "rpc_method", info.FullMethod)

		// panic recover
		defer func() {
			reqCtx.GetLog().Info("access", logFields...)
			if e := recover(); e != nil {
				err = fmt.Errorf("%s log_id = %s", base.ErrInternal, reqCtx.GetLog().GetLogId())
				reqCtx.GetLog().Error("Rpc server happen panic", "error", e)

				// stack
				stack := make([]byte, 8192)
				n := runtime.Stack(stack[:], false)
				log.Printf("%s Rpc server happen panic: %s", reqCtx.GetLog().GetLogId(), stack[:n])
			}
		}()

		// handle request
		// ??????err??????????????????????????????err?????????ecom.Error???????????????err?????????????????????????????????
		stdErr := base.ErrSuccess
		respRes, err = handler(ctx, req)
		if err != nil {
			stdErr = base.CastError(err)
		}
		// ????????????????????????header?????????????????????err=nil?????????Header.ErrCode??????
		respHeader := &pb.Header{
			Logid:    reqHeader.GetLogid(),
			FromNode: t.genTraceId(),
			Error:    t.convertErr(stdErr),
		}
		// ??????????????????header???response
		header := reflect.ValueOf(respRes).Elem().FieldByName("Header")
		if header.IsValid() && header.IsNil() && header.CanSet() {
			header.Set(reflect.ValueOf(respHeader))
		}

		// output ending log
		// ????????????log????????????SetInfoField?????????????????????ending log
		logFields = append(logFields, "status", stdErr.Status, "err_code", stdErr.Code,
			"err_msg", stdErr.Msg, "cost_time", reqCtx.GetTimer().Print())
		return respRes, err
	}
}

func (t *RpcServer) defReqHeader() *pb.Header {
	return &pb.Header{
		Logid:    utils.GenLogId(),
		FromNode: "",
		Error:    pb.XChainErrorEnum_UNKNOW_ERROR,
	}
}

func (t *RpcServer) createReqCtx(gctx context.Context, reqHeader *pb.Header) (sctx.ReqCtx, error) {
	// ???????????????ip
	clientIp, err := t.getClietIP(gctx)
	if err != nil {
		t.log.Error("access proc failed because get client ip failed", "error", err)
		return nil, fmt.Errorf("get client ip failed")
	}

	// ?????????????????????
	rctx, err := sctx.NewReqCtx(t.engine, reqHeader.GetLogid(), clientIp)
	if err != nil {
		t.log.Error("access proc failed because create request context failed", "error", err)
		return nil, fmt.Errorf("create request context failed")
	}

	return rctx, nil
}

func (t *RpcServer) getClietIP(gctx context.Context) (string, error) {
	pr, ok := peer.FromContext(gctx)
	if !ok {
		return "", fmt.Errorf("create peer form context failed")
	}

	if pr.Addr == nil || pr.Addr == net.Addr(nil) {
		return "", fmt.Errorf("get client_ip failed because peer.Addr is nil")
	}

	addrSlice := strings.Split(pr.Addr.String(), ":")
	return addrSlice[0], nil
}

// ??????????????????host??????????????????AES????????????????????????????????????
func (t *RpcServer) genTraceId() string {
	return utils.GetHostName()
}

// ????????????????????????????????????
func (t *RpcServer) convertErr(stdErr *base.Error) pb.XChainErrorEnum {
	if stdErr == nil {
		return pb.XChainErrorEnum_UNKNOW_ERROR
	}

	if errCode, ok := sctx.StdErrToXchainErrMap[stdErr.Code]; ok {
		return errCode
	}

	return pb.XChainErrorEnum_UNKNOW_ERROR
}
