package server

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"chatroom/internal/shared"
)

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	addr := conn.RemoteAddr()
	log.Printf("[INFO] New connection from %s", addr)

	// Wait for authentication
	msg, err := shared.ReadMessage(conn)
	if err != nil {
		log.Printf("[ERROR] Failed to read auth message from %s: %v", addr, err)
		return
	}

	log.Printf("[INFO] Received auth message from %s: %+v", addr, msg)

	if msg.Type != shared.TypeAuth {
		s.sendError(conn, "First message must be authentication")
		return
	}

	// Try to authenticate
	user, err := s.users.AuthenticateUser(msg.From, conn)
	if err != nil {
		s.sendError(conn, err.Error())
		return
	}

	// Send successful auth response
	s.sendAuthResponse(conn, true, "")

	// Broadcast user list update
	s.broadcastUserList()

	// Notify others about new user
	s.broadcastUserJoin(user.Username)

	// Message handling loop
	msgChan := make(chan *shared.Message)
	errChan := make(chan error)
	done := make(chan struct{})
	var messageWg sync.WaitGroup

	// Start message reader goroutine
	go func() {
		defer close(msgChan)
		defer close(errChan)

		for {
			select {
			case <-done:
				return
			default:
				msg, err := shared.ReadMessage(conn)
				if err != nil {
					errChan <- err
					return
				}
				msgChan <- msg
			}
		}
	}()

	// Cleanup function to ensure proper goroutine cleanup
	cleanup := func() {
		messageWg.Wait() // Wait for all message handlers to complete
		s.broadcastUserLeave(user.Username)
		s.users.Remove(user.Username)
		close(done) // Signal reader goroutine to stop
		s.broadcastUserList()
	}
	defer cleanup()

	// Handle messages
	for {
		select {
		case err, ok := <-errChan:
			if !ok {
				return
			}
			log.Printf("Error reading message from %s: %v", user.Username, err)
			return
		case msg, ok := <-msgChan:
			if !ok {
				return
			}
			msg.Timestamp = time.Now()

			// Process message in a separate goroutine
			messageWg.Add(1)
			go func(m *shared.Message) {
				defer messageWg.Done()
				if err := s.handleMessage(user, m); err != nil {
					log.Printf("Error handling message from %s: %v", user.Username, err)
				}
			}(msg)
		case <-s.done:
			return
		}
	}
}

func (s *Server) handleMessage(user *shared.User, msg *shared.Message) error {
	log.Printf("[DEBUG] Handling message from user %s, type: %s, content: %s",
		user.Username, msg.Type, msg.Content)

	msg.From = user.Username
	msg.Timestamp = time.Now()

	if strings.HasPrefix(msg.Content, "/w ") {
		log.Printf("[DEBUG] Processing private message from %s", user.Username)
		err := s.handlePrivateMessage(msg)
		if err != nil {
			log.Printf("[ERROR] Failed to handle private message from %s: %v", user.Username, err)
		}
		return err
	} else {
		log.Printf("[DEBUG] Broadcasting public message from %s", user.Username)
		err := s.broadcastPublicMessage(msg)
		if err != nil {
			log.Printf("[ERROR] Failed to broadcast message from %s: %v", user.Username, err)
		}
		return err
	}
}

func (s *Server) handlePrivateMessage(msg *shared.Message) error {
	log.Printf("[DEBUG] Processing private message: %+v", msg)

	// Parse "/w username message"
	parts := strings.SplitN(msg.Content, " ", 3)
	if len(parts) < 3 {
		log.Printf("[ERROR] Invalid whisper format from %s: %s", msg.From, msg.Content)
		s.sendError(msg.From, "Invalid whisper format. Use: /w username message")
		return fmt.Errorf("invalid whisper format")
	}

	targetUsername := parts[1]
	msg.Content = parts[2]
	msg.Type = shared.TypePrivate
	msg.To = targetUsername

	log.Printf("[DEBUG] Private message from %s to %s: %s",
		msg.From, targetUsername, msg.Content)

	// Find target user
	targetUser, exists := s.users.GetByUsername(targetUsername)
	if !exists {
		log.Printf("[ERROR] Target user not found: %s", targetUsername)
		s.sendError(msg.From, "User "+targetUsername+" not found")
		return fmt.Errorf("target user not found: %s", targetUsername)
	}

	// Send to target and sender
	if err := shared.WriteMessage(targetUser.Conn, msg); err != nil {
		log.Printf("[ERROR] Failed to send to target %s: %v", targetUsername, err)
		return fmt.Errorf("failed to send to target: %v", err)
	}
	log.Printf("[DEBUG] Message sent to target %s", targetUsername)

	if sender, exists := s.users.GetByUsername(msg.From); exists {
		if err := shared.WriteMessage(sender.Conn, msg); err != nil {
			log.Printf("[ERROR] Failed to send confirmation to sender %s: %v", msg.From, err)
			return fmt.Errorf("failed to send to sender: %v", err)
		}
		log.Printf("[DEBUG] Confirmation sent to sender %s", msg.From)
	}

	return nil
}

func (s *Server) broadcastPublicMessage(msg *shared.Message) error {
	log.Printf("[DEBUG] Broadcasting public message from %s: %s", msg.From, msg.Content)
	msg.Type = shared.TypePublic
	s.broadcast(msg)
	log.Printf("[DEBUG] Public message queued for broadcast")
	return nil
}

func (s *Server) broadcastUserList() {
	msg := &shared.Message{
		Type:      shared.TypeUserList,
		Users:     s.users.GetUsernames(),
		Timestamp: time.Now(),
	}
	s.broadcast(msg)
}

func (s *Server) broadcastUserJoin(username string) {
	msg := &shared.Message{
		Type:      shared.TypeJoin,
		Content:   username + " has joined the chat",
		Timestamp: time.Now(),
	}
	s.broadcast(msg)
}

func (s *Server) broadcastUserLeave(username string) {
	msg := &shared.Message{
		Type:      shared.TypeLeave,
		Content:   username + " has left the chat",
		Timestamp: time.Now(),
	}
	s.broadcast(msg)
}

func (s *Server) sendError(connOrUsername interface{}, errMsg string) {
	msg := &shared.Message{
		Type:      shared.TypeError,
		Content:   errMsg,
		Timestamp: time.Now(),
	}

	switch v := connOrUsername.(type) {
	case net.Conn:
		shared.WriteMessage(v, msg)
	case string:
		if user, exists := s.users.GetByUsername(v); exists {
			shared.WriteMessage(user.Conn, msg)
		}
	}
}

func (s *Server) sendAuthResponse(conn net.Conn, success bool, errorMsg string) {
	msg := &shared.Message{
		Type:      shared.TypeAuthResponse,
		Success:   success,
		Error:     errorMsg,
		Timestamp: time.Now(),
	}
	shared.WriteMessage(conn, msg)
}

func (s *Server) broadcast(msg *shared.Message) {
	log.Printf("[DEBUG] Attempting to broadcast message: Type=%s, From=%s, Content=%s",
		msg.Type, msg.From, msg.Content)

	select {
	case s.broadcastCh <- msg:
		log.Printf("[DEBUG] Message successfully queued for broadcast")
	default:
		log.Printf("[ERROR] Broadcast channel full, message dropped: %+v", msg)
	}
}
