// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.1
// source: network_service.proto

package protos

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// P2PServiceClient is the client API for P2PService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type P2PServiceClient interface {
	SendP2PMessage(ctx context.Context, opts ...grpc.CallOption) (P2PService_SendP2PMessageClient, error)
}

type p2PServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewP2PServiceClient(cc grpc.ClientConnInterface) P2PServiceClient {
	return &p2PServiceClient{cc}
}

func (c *p2PServiceClient) SendP2PMessage(ctx context.Context, opts ...grpc.CallOption) (P2PService_SendP2PMessageClient, error) {
	stream, err := c.cc.NewStream(ctx, &P2PService_ServiceDesc.Streams[0], "/protos.p2pService/SendP2pMessage", opts...)
	if err != nil {
		return nil, err
	}
	x := &p2PServiceSendP2PMessageClient{stream}
	return x, nil
}

type P2PService_SendP2PMessageClient interface {
	Send(*CoreMessage) error
	Recv() (*CoreMessage, error)
	grpc.ClientStream
}

type p2PServiceSendP2PMessageClient struct {
	grpc.ClientStream
}

func (x *p2PServiceSendP2PMessageClient) Send(m *CoreMessage) error {
	return x.ClientStream.SendMsg(m)
}

func (x *p2PServiceSendP2PMessageClient) Recv() (*CoreMessage, error) {
	m := new(CoreMessage)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// P2PServiceServer is the server API for P2PService service.
// All implementations should embed UnimplementedP2PServiceServer
// for forward compatibility
type P2PServiceServer interface {
	SendP2PMessage(P2PService_SendP2PMessageServer) error
}

// UnimplementedP2PServiceServer should be embedded to have forward compatible implementations.
type UnimplementedP2PServiceServer struct {
}

func (UnimplementedP2PServiceServer) SendP2PMessage(P2PService_SendP2PMessageServer) error {
	return status.Errorf(codes.Unimplemented, "method SendP2PMessage not implemented")
}

// UnsafeP2PServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to P2PServiceServer will
// result in compilation errors.
type UnsafeP2PServiceServer interface {
	mustEmbedUnimplementedP2PServiceServer()
}

func RegisterP2PServiceServer(s grpc.ServiceRegistrar, srv P2PServiceServer) {
	s.RegisterService(&P2PService_ServiceDesc, srv)
}

func _P2PService_SendP2PMessage_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(P2PServiceServer).SendP2PMessage(&p2PServiceSendP2PMessageServer{stream})
}

type P2PService_SendP2PMessageServer interface {
	Send(*CoreMessage) error
	Recv() (*CoreMessage, error)
	grpc.ServerStream
}

type p2PServiceSendP2PMessageServer struct {
	grpc.ServerStream
}

func (x *p2PServiceSendP2PMessageServer) Send(m *CoreMessage) error {
	return x.ServerStream.SendMsg(m)
}

func (x *p2PServiceSendP2PMessageServer) Recv() (*CoreMessage, error) {
	m := new(CoreMessage)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// P2PService_ServiceDesc is the grpc.ServiceDesc for P2PService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var P2PService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "protos.p2pService",
	HandlerType: (*P2PServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SendP2pMessage",
			Handler:       _P2PService_SendP2PMessage_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "network_service.proto",
}