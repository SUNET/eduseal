// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.21.12
// source: v1-validator.proto

package v1_validator

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Validator_Validate_FullMethodName = "/v1.validator.Validator/Validate"
)

// ValidatorClient is the client API for Validator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Validator validates documents
type ValidatorClient interface {
	Validate(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateReply, error)
}

type validatorClient struct {
	cc grpc.ClientConnInterface
}

func NewValidatorClient(cc grpc.ClientConnInterface) ValidatorClient {
	return &validatorClient{cc}
}

func (c *validatorClient) Validate(ctx context.Context, in *ValidateRequest, opts ...grpc.CallOption) (*ValidateReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ValidateReply)
	err := c.cc.Invoke(ctx, Validator_Validate_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ValidatorServer is the server API for Validator service.
// All implementations must embed UnimplementedValidatorServer
// for forward compatibility.
//
// Validator validates documents
type ValidatorServer interface {
	Validate(context.Context, *ValidateRequest) (*ValidateReply, error)
	mustEmbedUnimplementedValidatorServer()
}

// UnimplementedValidatorServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedValidatorServer struct{}

func (UnimplementedValidatorServer) Validate(context.Context, *ValidateRequest) (*ValidateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Validate not implemented")
}
func (UnimplementedValidatorServer) mustEmbedUnimplementedValidatorServer() {}
func (UnimplementedValidatorServer) testEmbeddedByValue()                   {}

// UnsafeValidatorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ValidatorServer will
// result in compilation errors.
type UnsafeValidatorServer interface {
	mustEmbedUnimplementedValidatorServer()
}

func RegisterValidatorServer(s grpc.ServiceRegistrar, srv ValidatorServer) {
	// If the following call pancis, it indicates UnimplementedValidatorServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Validator_ServiceDesc, srv)
}

func _Validator_Validate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ValidateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ValidatorServer).Validate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Validator_Validate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ValidatorServer).Validate(ctx, req.(*ValidateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Validator_ServiceDesc is the grpc.ServiceDesc for Validator service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Validator_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "v1.validator.Validator",
	HandlerType: (*ValidatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Validate",
			Handler:    _Validator_Validate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "v1-validator.proto",
}
