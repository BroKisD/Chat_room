package server

import (
	"net"

	"chatroom/internal/shared"
)

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Handle authentication
	user, err := s.auth.Authenticate(conn)
	if err != nil {
		return
	}

	// Add user to active users
	s.users.Add(user)
	defer s.users.Remove(user.ID)

	// Handle messages
	for {
		msg, err := shared.ReadMessage(conn)
		if err != nil {
			break
		}
		s.handleMessage(msg, user)
	}
}

func (s *Server) handleMessage(msg *shared.Message, sender *shared.User) {
	// Message handling implementation
}
