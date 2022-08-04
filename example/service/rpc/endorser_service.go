package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	sctx "github.com/wooyang2018/corechain/example/base"
	protos2 "github.com/wooyang2018/corechain/example/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	cryptoClient "github.com/wooyang2018/corechain/crypto/client"
	"github.com/wooyang2018/corechain/crypto/core/hash"
	engineBase "github.com/wooyang2018/corechain/engines/base"
	scom "github.com/wooyang2018/corechain/example/service/common"
	sconf "github.com/wooyang2018/corechain/example/service/config"
	"github.com/wooyang2018/corechain/state/txhash"
)

const (
	EndorserModuleDefault = "default"
	EndorserModuleProxy   = "proxy"
)

type XEndorser interface {
	EndorserCall(gctx context.Context, req *protos2.EndorserRequest) (*protos2.EndorserResponse, error)
}

type ProxyXEndorser struct {
	engine      engineBase.Engine
	clientCache sync.Map
	mutex       sync.Mutex
	conf        *sconf.ServConf
}

func newEndorserService(cfg *sconf.ServConf, engine engineBase.Engine, svr XEndorserServer) (XEndorser, error) {
	switch cfg.EndorserModule {
	case EndorserModuleDefault:
		dxe := NewDefaultXEndorser(svr, engine)
		return dxe, nil
	case EndorserModuleProxy:
		return &ProxyXEndorser{
			engine: engine,
			conf:   cfg,
		}, nil
	default:
		return nil, fmt.Errorf("unknown endorser module")
	}

}

func (pxe *ProxyXEndorser) EndorserCall(gctx context.Context, req *protos2.EndorserRequest) (*protos2.EndorserResponse, error) {
	resp := &protos2.EndorserResponse{}
	rctx := sctx.ValueReqCtx(gctx)
	endc, err := pxe.getClient(pxe.getHost())
	if err != nil {
		return resp, err
	}
	res, err := endc.EndorserCall(gctx, req)
	if err != nil {
		return resp, err
	}
	resp.EndorserAddress = res.EndorserAddress
	resp.ResponseName = res.ResponseName
	resp.ResponseData = res.ResponseData
	resp.EndorserSign = res.EndorserSign
	rctx.GetLog().SetInfoField("bc_name", req.GetBcName())
	rctx.GetLog().SetInfoField("request_name", req.GetBcName())
	return resp, nil
}

func (pxe *ProxyXEndorser) getHost() string {
	host := ""
	hostCnt := len(pxe.conf.EndorserHosts)
	if hostCnt > 0 {
		rand.Seed(time.Now().Unix())
		index := rand.Intn(hostCnt)
		host = pxe.conf.EndorserHosts[index]
	}
	return host
}

func (pxe *ProxyXEndorser) getClient(host string) (protos2.XendorserClient, error) {
	if host == "" {
		return nil, fmt.Errorf("empty host")
	}
	if c, ok := pxe.clientCache.Load(host); ok {
		return c.(protos2.XendorserClient), nil
	}

	pxe.mutex.Lock()
	defer pxe.mutex.Unlock()
	if c, ok := pxe.clientCache.Load(host); ok {
		return c.(protos2.XendorserClient), nil
	}
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	c := protos2.NewXendorserClient(conn)
	pxe.clientCache.Store(host, c)
	return c, nil
}

type XEndorserServer interface {
	// PostTx post Transaction to a node
	PostTx(context.Context, *protos2.TxStatus) (*protos2.CommonReply, error)
	// QueryTx query Transaction by TxStatus,
	// Bcname and Txid are required for this
	QueryTx(context.Context, *protos2.TxStatus) (*protos2.TxStatus, error)
	// PreExecWithSelectUTXO preExec & selectUtxo
	PreExecWithSelectUTXO(context.Context, *protos2.PreExecWithSelectUTXORequest) (*protos2.PreExecWithSelectUTXOResponse, error)
	// 预执行合约
	PreExec(context.Context, *protos2.InvokeRPCRequest) (*protos2.InvokeRPCResponse, error)
}

type DefaultXEndorser struct {
	svr         XEndorserServer
	requestType map[string]bool
	engine      engineBase.Engine
}

var _ XEndorser = (*DefaultXEndorser)(nil)

const (
	// DefaultKeyPath is the default key path
	DefaultKeyPath = "./data/endorser/keys/"
)

func NewDefaultXEndorser(svr XEndorserServer, engine engineBase.Engine) *DefaultXEndorser {
	return &DefaultXEndorser{
		requestType: map[string]bool{
			"PreExecWithFee":    true,
			"ComplianceCheck":   true,
			"CrossQueryPreExec": true,
			"TxQuery":           true,
		},
		svr:    svr,
		engine: engine,
	}
}

// EndorserCall process endorser call
func (dxe *DefaultXEndorser) EndorserCall(ctx context.Context, req *protos2.EndorserRequest) (*protos2.EndorserResponse, error) {
	// make response header
	resHeader := &protos2.Header{
		Error: protos2.XChainErrorEnum_SUCCESS,
	}
	if req.GetHeader() == nil {
		resHeader.Logid = req.GetHeader().GetLogid()
	}

	// check param
	if _, ok := dxe.requestType[req.GetRequestName()]; !ok {
		resHeader.Error = protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR
		return dxe.generateErrorResponse(req, resHeader, errors.New("request name not supported"))
	}

	reqCtx, err := dxe.createReqCtx(ctx, req.Header)
	if err != nil {
		return nil, err
	}
	ctx = sctx.WithReqCtx(ctx, reqCtx)

	switch req.GetRequestName() {
	case "ComplianceCheck":
		success, errcode, err := dxe.processFee(ctx, req)
		if err != nil || !success {
			resHeader.Error = errcode
			return dxe.generateErrorResponse(req, resHeader, err)
		}
		addr, sign, err := dxe.generateTxSign(ctx, req)
		if err != nil {
			resHeader.Error = protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR
			return dxe.generateErrorResponse(req, resHeader, err)
		}

		reply := &protos2.CommonReply{
			Header: &protos2.Header{
				Error: protos2.XChainErrorEnum_SUCCESS,
			},
		}
		resData, err := json.Marshal(reply)
		if err != nil {
			resHeader.Error = protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR
			return dxe.generateErrorResponse(req, resHeader, err)
		}
		return dxe.generateSuccessResponse(req, resData, addr, sign, resHeader)

	case "PreExecWithFee":
		resData, errcode, err := dxe.getPreExecResult(ctx, req)
		if err != nil {
			resHeader.Error = errcode
			return dxe.generateErrorResponse(req, resHeader, err)
		}
		return dxe.generateSuccessResponse(req, resData, nil, nil, resHeader)

	case "CrossQueryPreExec":
		resData, errcode, err := dxe.getCrossQueryResult(ctx, req)
		if err != nil {
			resHeader.Error = errcode
			return dxe.generateErrorResponse(req, resHeader, err)
		}
		data := append(req.RequestData[:], resData[:]...)
		digest := hash.HashUsingSha256(data)
		addr, sign, err := dxe.signData(ctx, digest, DefaultKeyPath)
		if err != nil {
			resHeader.Error = errcode
			return dxe.generateErrorResponse(req, resHeader, err)
		}
		return dxe.generateSuccessResponse(req, resData, addr, sign, resHeader)
	case "TxQuery":
		resData, errcode, err := dxe.getTxResult(ctx, req)
		if err != nil {
			resHeader.Error = errcode
			return dxe.generateErrorResponse(req, resHeader, err)
		}
		data := append(req.RequestData[:], resData[:]...)
		digest := hash.HashUsingSha256(data)
		addr, sign, err := dxe.signData(ctx, digest, DefaultKeyPath)
		if err != nil {
			resHeader.Error = errcode
			return dxe.generateErrorResponse(req, resHeader, err)
		}
		return dxe.generateSuccessResponse(req, resData, addr, sign, resHeader)
	}

	return nil, nil
}

func (dxe *DefaultXEndorser) getPreExecResult(ctx context.Context, req *protos2.EndorserRequest) ([]byte, protos2.XChainErrorEnum, error) {
	request := &protos2.PreExecWithSelectUTXORequest{}
	err := json.Unmarshal(req.GetRequestData(), request)
	if err != nil {
		return nil, protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR, err
	}

	res, err := dxe.svr.PreExecWithSelectUTXO(ctx, request)
	if err != nil {
		return nil, res.GetHeader().GetError(), err
	}

	sData, err := json.Marshal(res)
	if err != nil {
		return nil, protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR, err
	}
	return sData, protos2.XChainErrorEnum_SUCCESS, nil
}

func (dxe *DefaultXEndorser) getCrossQueryResult(ctx context.Context, req *protos2.EndorserRequest) ([]byte, protos2.XChainErrorEnum, error) {
	cqReq := &protos2.CrossQueryRequest{}
	err := json.Unmarshal(req.GetRequestData(), cqReq)
	if err != nil {
		return nil, protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR, err
	}

	preExecReq := &protos2.InvokeRPCRequest{
		Header:      req.GetHeader(),
		Bcname:      cqReq.GetBcname(),
		Initiator:   cqReq.GetInitiator(),
		AuthRequire: cqReq.GetAuthRequire(),
	}
	preExecReq.Requests = append(preExecReq.Requests, cqReq.GetRequest())

	preExecRes, err := dxe.svr.PreExec(ctx, preExecReq)
	if err != nil {
		return nil, preExecRes.GetHeader().GetError(), err
	}

	if preExecRes.GetHeader().GetError() != protos2.XChainErrorEnum_SUCCESS {
		return nil, preExecRes.GetHeader().GetError(), errors.New("PreExec not success")
	}

	res := &protos2.CrossQueryResponse{}
	contractRes := preExecRes.GetResponse().GetResponses()
	if len(contractRes) > 0 {
		res.Response = contractRes[len(contractRes)-1]
	}

	sData, err := json.Marshal(res)
	if err != nil {
		return nil, protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR, err
	}

	return sData, protos2.XChainErrorEnum_SUCCESS, nil
}

func (dxe *DefaultXEndorser) getTxResult(ctx context.Context, req *protos2.EndorserRequest) ([]byte, protos2.XChainErrorEnum, error) {
	request := &protos2.TxStatus{}
	err := json.Unmarshal(req.GetRequestData(), request)
	if err != nil {
		return nil, protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR, err
	}

	reply, err := dxe.svr.QueryTx(ctx, request)
	if err != nil {
		return nil, reply.GetHeader().GetError(), err
	}

	if reply.GetHeader().GetError() != protos2.XChainErrorEnum_SUCCESS {
		return nil, reply.GetHeader().GetError(), errors.New("QueryTx not success")
	}

	if reply.Tx == nil {
		return nil, reply.GetHeader().GetError(), errors.New("tx not found")
	}

	sData, err := json.Marshal(reply.Tx)
	if err != nil {
		return nil, protos2.XChainErrorEnum_SERVICE_REFUSED_ERROR, err
	}

	return sData, protos2.XChainErrorEnum_SUCCESS, nil
}

func (dxe *DefaultXEndorser) processFee(ctx context.Context, req *protos2.EndorserRequest) (bool, protos2.XChainErrorEnum, error) {
	if req.GetFee() == nil {
		// no fee provided, default to true
		return true, protos2.XChainErrorEnum_SUCCESS, nil
	}

	txStatus := &protos2.TxStatus{
		Txid:   req.GetFee().GetTxid(),
		Bcname: req.GetBcName(),
		Tx:     req.GetFee(),
	}

	res, err := dxe.svr.PostTx(ctx, txStatus)
	if err != nil {
		return false, res.GetHeader().GetError(), err
	} else if res.GetHeader().GetError() != protos2.XChainErrorEnum_SUCCESS {
		return false, res.GetHeader().GetError(), errors.New("Fee post to chain failed")
	}

	return true, protos2.XChainErrorEnum_SUCCESS, nil
}

func (dxe *DefaultXEndorser) generateTxSign(ctx context.Context, req *protos2.EndorserRequest) ([]byte, *protos2.SignatureInfo, error) {
	if req.GetRequestData() == nil {
		return nil, nil, errors.New("request data is empty")
	}

	txStatus := &protos2.TxStatus{}
	err := json.Unmarshal(req.GetRequestData(), txStatus)
	if err != nil {
		return nil, nil, err
	}

	tx := scom.TxToXledger(txStatus.GetTx())
	digest, err := txhash.MakeTxDigestHash(tx)
	if err != nil {
		return nil, nil, err
	}

	return dxe.signData(ctx, digest, DefaultKeyPath)
}

func (dxe *DefaultXEndorser) signData(ctx context.Context, data []byte, keypath string) ([]byte, *protos2.SignatureInfo, error) {
	addr, jsonSKey, jsonAKey, err := dxe.getEndorserKey(keypath)
	if err != nil {
		return nil, nil, err
	}

	cryptoClient, err := cryptoClient.CreateCryptoClientFromJSONPrivateKey(jsonSKey)
	if err != nil {
		return nil, nil, err
	}

	privateKey, err := cryptoClient.GetEcdsaPrivateKeyFromJsonStr(string(jsonSKey))
	if err != nil {
		return nil, nil, err
	}

	sign, err := cryptoClient.SignECDSA(privateKey, data)
	if err != nil {
		return nil, nil, err
	}

	signInfo := &protos2.SignatureInfo{
		PublicKey: string(jsonAKey),
		Sign:      sign,
	}
	return addr, signInfo, nil
}

func (dxe *DefaultXEndorser) generateErrorResponse(req *protos2.EndorserRequest, header *protos2.Header,
	err error) (*protos2.EndorserResponse, error) {
	res := &protos2.EndorserResponse{
		Header:       header,
		ResponseName: req.GetRequestName(),
	}
	return res, err
}

func (dxe *DefaultXEndorser) generateSuccessResponse(req *protos2.EndorserRequest, resData []byte,
	addr []byte, sign *protos2.SignatureInfo, header *protos2.Header) (*protos2.EndorserResponse, error) {
	res := &protos2.EndorserResponse{
		Header:          header,
		ResponseName:    req.GetRequestName(),
		ResponseData:    resData,
		EndorserAddress: string(addr),
		EndorserSign:    sign,
	}
	return res, nil
}

func (dxe *DefaultXEndorser) getEndorserKey(keypath string) ([]byte, []byte, []byte, error) {
	sk, err := ioutil.ReadFile(keypath + "private.key")
	if err != nil {
		return nil, nil, nil, err
	}

	ak, err := ioutil.ReadFile(keypath + "public.key")
	if err != nil {
		return nil, nil, nil, err
	}

	addr, err := ioutil.ReadFile(keypath + "address")
	return addr, sk, ak, err
}

func (dxe *DefaultXEndorser) createReqCtx(gctx context.Context, reqHeader *protos2.Header) (sctx.ReqCtx, error) {
	// 获取客户端ip
	clientIp, err := dxe.getClietIP(gctx)
	if err != nil {
		return nil, fmt.Errorf("get client ip failed.err:%v", err)
	}

	// 创建请求上下文
	rctx, err := sctx.NewReqCtx(dxe.engine, reqHeader.GetLogid(), clientIp)
	if err != nil {
		return nil, fmt.Errorf("create request context failed.err:%v", err)
	}

	return rctx, nil
}

func (dxe *DefaultXEndorser) getClietIP(gctx context.Context) (string, error) {
	pr, ok := peer.FromContext(gctx)
	if !ok {
		return "", nil
	}

	if pr.Addr == nil || pr.Addr == net.Addr(nil) {
		return "", fmt.Errorf("get client_ip failed because peer.Addr is nil")
	}

	addrSlice := strings.Split(pr.Addr.String(), ":")
	return addrSlice[0], nil
}
