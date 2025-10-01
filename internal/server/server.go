package server

import (
	"context"
	"log"
	"net"
	"sync"

	"chatroom/internal/server/auth"
	"chatroom/internal/server/users"
	"chatroom/internal/shared"
)

type Server struct {
	listener    net.Listener
	addr        string
	auth        *auth.Authenticator
	users       *users.Manager
	mu          sync.RWMutex
	broadcastCh chan *shared.Message
	done        chan struct{}
	connections sync.WaitGroup
}

func New(addr string) *Server {
	return &Server{
		addr:        addr,
		auth:        auth.New(),
		users:       users.New(),
		broadcastCh: make(chan *shared.Message, 100), // Buffer for 100 messages
		done:        make(chan struct{}),
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = listener

	// Start broadcast handler
	go s.handleBroadcasts()

	return s.serve()
}

// Gracefully shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Signal shutdown
	close(s.done)

	// Close listener to stop accepting new connections
	if err := s.listener.Close(); err != nil {
		return err
	}

	// Wait for all connections to finish with context timeout
	done := make(chan struct{})
	go func() {
		s.connections.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (s *Server) serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return nil // Normal shutdown
			default:
				return err // Unexpected error
			}
		}

		s.connections.Add(1)
		go func(conn net.Conn) {
			defer s.connections.Done()
			s.handleConnection(conn)
		}(conn)
	}
}

func (s *Server) handleBroadcasts() {
	for {
		select {
		case <-s.done:
			return
		case msg := <-s.broadcastCh:
			s.mu.RLock()
			users := s.users.GetAll()
			s.mu.RUnlock()

			for _, user := range users {
				if user.Conn != nil {
					// Use goroutine to prevent slow clients from blocking broadcasts
					go func(u *shared.User, m *shared.Message) {
						if err := shared.WriteMessage(u.Conn, m); err != nil {
							log.Printf("Error broadcasting to %s: %v", u.Username, err)
						}
					}(user, msg)
				}
			}
		}
	}
}

// func (s *Server) handleConnection(conn net.Conn) {
// 	// Implementation handled in handler.go
// }
