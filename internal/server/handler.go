package server

import (
	"log"
	"net"

	"chatroom/internal/shared"
)

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Printf("New connection from %s", conn.RemoteAddr())

	// Handle messages
	for {
		msg, err := shared.ReadMessage(conn)
		if err != nil {
			log.Printf("Error reading message from %s: %v", conn.RemoteAddr(), err)
			break
		}

		go s.handleMessage(conn, msg)
	}
}

func (s *Server) handleMessage(conn net.Conn, msg *shared.Message) {
	switch msg.Type {
	case shared.MessageTypeChat:
		// Broadcast message to all connected clients
		s.broadcast(msg)
	case shared.MessageTypeJoin:
		// Handle new user joining
		s.handleJoin(conn, msg)
	case shared.MessageTypeLeave:
		// Handle user leaving
		s.handleLeave(msg)
	}
}

func (s *Server) broadcast(msg *shared.Message) {
	select {
	case s.broadcastCh <- msg:
		// Message sent successfully
	default:
		log.Println("Broadcast channel full, message dropped")
	}
}

func (s *Server) handleJoin(conn net.Conn, msg *shared.Message) {
	user := &shared.User{
		Username: msg.Sender,
		Conn:     conn,
	}
	s.users.Add(user)

	// Notify others that a new user has joined
	broadcast := &shared.Message{
		Type:    shared.MessageTypeJoin,
		Content: msg.Sender + " has joined the chat",
	}
	s.broadcast(broadcast)
}

func (s *Server) handleLeave(msg *shared.Message) {
	s.users.RemoveByUsername(msg.Sender)

	// Notify others that a user has left
	broadcast := &shared.Message{
		Type:    shared.MessageTypeLeave,
		Content: msg.Sender + " has left the chat",
	}
	s.broadcast(broadcast)
}
