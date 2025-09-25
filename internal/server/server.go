package server

import (
	"net"
	"sync"

	"chatroom/internal/server/auth"
	"chatroom/internal/server/users"
)

type Server struct {
	listener net.Listener
	addr     string
	auth     *auth.Authenticator
	users    *users.Manager
	mu       sync.RWMutex
}

func New(addr string) *Server {
	return &Server{
		addr:  addr,
		auth:  auth.New(),
		users: users.New(),
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = listener
	return s.serve()
}

func (s *Server) serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	// Implementation handled in handler.go
}
