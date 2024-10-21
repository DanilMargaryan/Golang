package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
)

func authInterceptor(serve *Serve) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		if err := serve.authInterceptor(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func logsInterceptor(serve *Serve) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		resp, err = handler(ctx, req)
		if err != nil {
			return resp, err
		}
		if err = serve.logsInterceptor(ctx); err != nil {
			return resp, err
		}
		return resp, nil
	}
}

func streamAuthInterceptor(serve *Serve) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if err := serve.authInterceptor(ss.Context()); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func streamLogInterceptor(serve *Serve) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		go func() {
			_ = serve.logsInterceptor(ss.Context())
		}()
		if err := handler(srv, ss); err != nil {
			return err
		}
		return nil
	}
}

func StartMyMicroservice(ctx context.Context, listenAddr string, aclData string) error {
	var accessUserMap ACL
	var logChanel = make(chan *Event)
	err := json.Unmarshal([]byte(aclData), &accessUserMap)
	if err != nil {
		return err
	}

	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	serve := Serve{
		accessUserMap: accessUserMap,
		logChannel:    logChanel,
		listeners:     &[]Admin_LoggingServer{},
		stats:         make(map[Admin_StatisticsServer]*Stat),
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			authInterceptor(&serve),
			logsInterceptor(&serve),
		),
		grpc.ChainStreamInterceptor(
			streamAuthInterceptor(&serve),
			streamLogInterceptor(&serve),
		),
	)
	bizSrv := BizNew(&serve)
	adminSrv := AdminNew(&serve)

	RegisterBizServer(server, bizSrv)
	RegisterAdminServer(server, adminSrv)

	go func() {
		if err := server.Serve(l); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				server.GracefulStop()
				return
			case event := <-serve.logChannel:
				for _, server := range *serve.listeners {
					err = server.Send(event)
				}
			}
		}

	}()

	return nil
}
