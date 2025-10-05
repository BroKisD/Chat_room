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
	roomKey     []byte
}

func New(addr string) *Server {
	return &Server{
		addr:        addr,
		auth:        auth.New(),
		users:       users.New(),
		broadcastCh: make(chan *shared.Message, 100), // Buffer for 100 messages
		done:        make(chan struct{}),
		roomKey:     shared.GenerateRoomKey(),
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
	// WaitGroup to track message sending goroutines
	var wg sync.WaitGroup

	for {
		select {
		case <-s.done:
			wg.Wait() // Wait for all broadcasts to complete before shutting down
			return
		case msg := <-s.broadcastCh:
			log.Printf("[BROADCAST] New message: %+v", msg)

			s.mu.RLock()
			users := s.users.GetAll()
			s.mu.RUnlock()

			// Start a broadcast batch
			wg.Add(len(users))

			for _, user := range users {
				if user.Conn != nil {
					go func(u *shared.User, m *shared.Message) {
						defer wg.Done()
						if err := shared.WriteMessage(u.Conn, m); err != nil {
							log.Printf("Error broadcasting to %s: %v", u.Username, err)
						}
					}(user, msg)
				} else {
					wg.Done() // Don't forget to decrease counter for nil connections
					log.Printf("[WARN] User %s has a nil connection, skipping broadcast", user.Username)
				}
			}
		}
	}
}
