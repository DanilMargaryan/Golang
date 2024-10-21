package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"strings"
	"sync"
)

type ACL map[string][]string

// Serve хранит данные микросервиса
type Serve struct {
	accessUserMap ACL                              // ACL (правила доступа)
	logChannel    chan *Event                      // Канал для логов
	listeners     *[]Admin_LoggingServer           // Подписчики на логи
	mu            sync.Mutex                       // Мьютекс для защиты данных
	stats         map[Admin_StatisticsServer]*Stat // Статистика по клиентам
}

func (s *Serve) authInterceptor(ctx context.Context) error {
	methodName, ok := grpc.Method(ctx)
	if !ok {
		return status.Error(codes.Unknown, "Unknown method")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "metadata not found in context")
	}

	consumers := md.Get("consumer")
	if len(consumers) == 0 {
		return status.Error(codes.Unauthenticated, "metadata not found in context")
	}
	consumer := consumers[0]

	allowedMethods, exists := s.accessUserMap[consumer]
	if !exists {
		return status.Errorf(codes.Unauthenticated, "access denied for consumer %s", consumer)
	}

	for _, method := range allowedMethods {
		if method == methodName || strings.HasSuffix(method, "/*") {
			return nil
		}
	}

	return status.Error(codes.Unauthenticated,
		"access method not found for this consumer")
}

func (s *Serve) logsInterceptor(ctx context.Context) error {
	methodName, ok := grpc.Method(ctx)
	if !ok {
		return status.Error(codes.Unknown, "Unknown method")
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "metadata not found in context")
	}

	p, ok := peer.FromContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "remove address not found")
	}

	consumers := md.Get("consumer")
	if len(consumers) == 0 {
		return status.Error(codes.Unauthenticated, "metadata not found in context")
	}
	consumer := consumers[0]

	for _, value := range s.stats {
		value.ByConsumer[consumer]++
		value.ByMethod[methodName]++
	}

	event := &Event{
		Consumer: consumer,
		Method:   methodName,
		Host:     p.Addr.String(),
	}
	s.logChannel <- event

	return nil
}
