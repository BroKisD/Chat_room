package server

import (
	"bufio"
	"context"
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

	reader := bufio.NewReader(conn)
	msg, err := shared.ReadMessage(reader)

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

	// Notify others about new users
	s.broadcastUserJoin(user.Username)
	log.Printf("[INFO] User %s joined from %s", user.Username, addr)

	// Broadcast user list update
	s.broadcastUserList()
	log.Printf("[INFO] Sent user list to %s", user.Username)

	msgChan := make(chan *shared.Message, 100) // Buffered to prevent blocking
	errChan := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	var messageWg sync.WaitGroup

	// Start message reader goroutine
	go func() {
		defer close(msgChan)
		defer close(errChan)
		reader := bufio.NewReader(conn) // thêm dòng này
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
			msg, err := shared.ReadMessage(reader) // dùng reader
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				errChan <- err
				return
			}
			msgChan <- msg
		}
	}()

	// Cleanup function
	cleanup := func() {
		cancel()         // Signal reader to stop
		messageWg.Wait() // Wait for message handlers
		s.broadcastUserLeave(user.Username)
		s.users.Remove(user.Username)
		s.broadcastUserList()
	}
	defer cleanup()

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
		err := s.handlePrivateMessage(user, msg) // Pass user object
		if err != nil {
			log.Printf("[ERROR] Failed to handle private message from %s: %v", user.Username, err)
		}
		return err
	} else if msg.Type == shared.TypePublic {
		log.Printf("[DEBUG] Broadcasting public message from %s", user.Username)
		err := s.broadcastPublicMessage(msg)
		if err != nil {
			log.Printf("[ERROR] Failed to broadcast message from %s: %v", user.Username, err)
		}
		return err
	} else {
		log.Printf("[WARN] Unknown message type %s from %s", msg.Type, user.Username)
		return nil
	}
}

func (s *Server) handlePrivateMessage(user *shared.User, msg *shared.Message) error {
	log.Printf("[DEBU] Processing private message: %+v", msg)

	parts := strings.SplitN(msg.Content, " ", 3)
	if len(parts) < 3 {
		log.Printf("[ERROR] Invalid whisper format from %s: %s", msg.From, msg.Content)
		// Use the user's connection directly instead of lookup
		s.sendErrorToConn(user.Conn, "Invalid whisper format. Use: /w username message")
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
		s.sendErrorToConn(user.Conn, "User "+targetUsername+" not found")
		return fmt.Errorf("target user not found: %s", targetUsername)
	}

	// Use thread-safe write
	if err := targetUser.WriteMessage(msg); err != nil {
		log.Printf("[ERROR] Failed to send to target %s: %v", targetUsername, err)
		return fmt.Errorf("failed to send to target: %v", err)
	}
	log.Printf("[DEBUG] Message sent to target %s", targetUsername)

	// Send to sender using their connection directly
	if err := user.WriteMessage(msg); err != nil {
		log.Printf("[ERROR] Failed to send confirmation to sender %s: %v", msg.From, err)
		return fmt.Errorf("failed to send to sender: %v", err)
	}
	log.Printf("[DEBUG] Confirmation sent to sender %s", msg.From)

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

func (s *Server) sendErrorToConn(conn net.Conn, errMsg string) {
	msg := &shared.Message{
		Type:      shared.TypeError,
		Content:   errMsg,
		Timestamp: time.Now(),
	}
	shared.WriteMessage(conn, msg)
}
