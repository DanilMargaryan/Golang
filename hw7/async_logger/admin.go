package main

import "time"

type GrpcAdminServer struct {
	*Serve
}

func AdminNew(s *Serve) *GrpcAdminServer {
	return &GrpcAdminServer{s}
}

func (s *GrpcAdminServer) removeListener(stream Admin_LoggingServer) {
	for i, listener := range *s.listeners {
		if listener == stream {
			s.mu.Lock()
			*s.listeners = append((*s.listeners)[:i], (*s.listeners)[i+1:]...)
			s.mu.Unlock()
			break
		}
	}
}

func (s *GrpcAdminServer) Logging(_ *Nothing, stream Admin_LoggingServer) error {
	ctx := stream.Context()

	time.Sleep(1)

	s.mu.Lock()
	*s.listeners = append(*s.listeners, stream)
	s.mu.Unlock()

	defer s.removeListener(stream)

	<-ctx.Done()
	return stream.Context().Err()
}

func (s *GrpcAdminServer) Statistics(interval *StatInterval, stream Admin_StatisticsServer) error {
	ctx := stream.Context()
	timer := time.NewTimer(time.Duration(interval.IntervalSeconds) * time.Second)
	stat := &Stat{
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}

	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			delete(s.stats, stream)
			s.mu.Unlock()
			return stream.Context().Err()
		case <-timer.C:
			err := stream.Send(&Stat{
				Timestamp:  time.Now().Unix(),
				ByMethod:   stat.ByMethod,
				ByConsumer: stat.ByConsumer,
			})
			if err != nil {
				return err
			}
			stat.ByMethod = make(map[string]uint64)
			stat.ByConsumer = make(map[string]uint64)
			timer.Reset(time.Duration(interval.IntervalSeconds) * time.Second)
		}
	}

	return nil
}

func (s *GrpcAdminServer) mustEmbedUnimplementedAdminServer() {
	panic("implement me")
}
