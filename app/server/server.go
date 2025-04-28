package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/xtls/xray-core/common"
)

type Config struct {
	Path string
}

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, cfg any) (any, error) {
		return New(ctx, cfg.(*Config))
	}))
}

type Server struct {
	ctx      context.Context
	listener net.Listener
	conns    map[net.Conn]struct{}
	mutex    sync.Mutex
}

func New(ctx context.Context, cfg *Config) (*Server, error) {
	if _, err := os.Stat(cfg.Path); err == nil {
		if err := os.Remove(cfg.Path); err != nil {
			return nil, fmt.Errorf("failed to remove existing socket: %v", err)
		}
	}
	listener, err := net.Listen("unix", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to create unix socket: %v", err)
	}
	return &Server{
		ctx:      ctx,
		listener: listener,
		conns:    make(map[net.Conn]struct{}),
	}, nil
}

func (s *Server) Type() any {
	return (*Server)(nil)
}

func (s *Server) Start() error {
	go s.acceptLoop()
	return nil
}

func (s *Server) Close() error {
	s.mutex.Lock()
	for conn := range s.conns {
		conn.Close()
		delete(s.conns, conn)
	}
	s.mutex.Unlock()
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("failed to close listener: %v", err)
	}
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mutex.Lock()
		s.conns[conn] = struct{}{}
		s.mutex.Unlock()
	}
}
