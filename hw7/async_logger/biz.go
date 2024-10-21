package main

import (
	"context"
)

type GrpcBizServer struct {
	*Serve
}

func (b *GrpcBizServer) mustEmbedUnimplementedBizServer() {
	panic("implement me")
}

func BizNew(s *Serve) *GrpcBizServer {
	return &GrpcBizServer{s}
}

func (b *GrpcBizServer) Check(ctx context.Context, in *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (b *GrpcBizServer) Add(ctx context.Context, in *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (b *GrpcBizServer) Test(ctx context.Context, in *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}
