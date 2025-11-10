package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"chatroom/internal/server/filetransfer"
	"chatroom/internal/server/users"
	"chatroom/internal/shared"
)

type Server struct {
	listener     net.Listener
	addr         string
	users        *users.Manager
	mu           sync.RWMutex
	broadcastCh  chan *shared.Message
	done         chan struct{}
	connections  sync.WaitGroup
	roomKey      []byte
	stateFile    string
	fileTransfer *filetransfer.FileTransfer
}

func New(addr string) *Server {
	uploadDir := "uploads"
	os.MkdirAll(uploadDir, 0755)

	s := &Server{
		addr:         addr,
		users:        users.New(),
		broadcastCh:  make(chan *shared.Message, 100),
		done:         make(chan struct{}),
		stateFile:    "server_state.json",
		fileTransfer: filetransfer.New(uploadDir),
	}

	s.loadOrGenerateRoomKey()

	// Try loading saved state
	if err := s.LoadState(); err == nil {
		log.Println("[INFO] Server state loaded from file.")
	} else {
		log.Println("[WARN] No previous state found, generating new room key.")
	}

	return s
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

func (s *Server) Shutdown(ctx context.Context) error {

	log.Println("[INFO] Saving server state before shutdown...")

	if err := s.SaveState(); err != nil {
		log.Printf("[ERROR] Failed to save server state: %v", err)
	}

	close(s.done)
	s.users.Clear()

	if err := s.listener.Close(); err != nil {
		return err
	}

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

func (s *Server) SaveState() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state := map[string]interface{}{
		"roomKey": base64.StdEncoding.EncodeToString(s.roomKey),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.stateFile, data, 0644)
}

func (s *Server) LoadState() error {
	data, err := os.ReadFile(s.stateFile)
	if err != nil {
		return err
	}

	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	if rk, ok := state["roomKey"].(string); ok && rk != "" {
		decoded, err := base64.StdEncoding.DecodeString(rk)
		if err != nil {
			log.Printf("[ERROR] Failed to decode room key: %v", err)
			s.roomKey = shared.GenerateRoomKey()
		} else if len(decoded) == 0 {
			log.Printf("[WARN] Loaded empty room key, regenerating new one")
			s.roomKey = shared.GenerateRoomKey()
		} else {
			s.roomKey = decoded
		}
	} else {
		log.Printf("[WARN] No valid room key found in state, generating new one")
		s.roomKey = shared.GenerateRoomKey()
	}
	return nil
}

func (s *Server) loadOrGenerateRoomKey() {
	data, err := os.ReadFile("room.key")
	if err == nil {
		str := strings.TrimSpace(string(data))
		decoded, err := base64.StdEncoding.DecodeString(str)
		if err == nil && len(decoded) == 32 {
			s.roomKey = decoded
			log.Printf("[INFO] Loaded existing room key (len=%d)", len(s.roomKey))
			return
		}
		log.Printf("[WARN] Invalid room key format, regenerating new key: %v", err)
	}

	// Generate new key
	s.roomKey = shared.GenerateRoomKey()
	if s.roomKey == nil {
		log.Fatal("[FATAL] Failed to generate room key")
	}
	encoded := base64.StdEncoding.EncodeToString(s.roomKey)
	if err := os.WriteFile("room.key", []byte(encoded), 0600); err != nil {
		log.Printf("[ERROR] Failed to save room key: %v", err)
	}
	log.Printf("[INFO] Generated new room key (len=%d)", len(s.roomKey))
}
