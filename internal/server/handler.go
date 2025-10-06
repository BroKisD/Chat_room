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
		reader := bufio.NewReader(conn)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
			msg, err := shared.ReadMessage(reader)
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

	if msg.Type == shared.TypePublicKeyRequest {
		log.Printf("[DEBUG] Handling public key request from %s for %s",
			user.Username, msg.To)
		err := s.handlePublicKeyRequest(msg)
		if err != nil {
			log.Printf("[ERROR] Failed to handle public key request from %s: %v",
				user.Username, err)
		}
		return err
	}

	if msg.Type == shared.TypePublicKey {

		log.Printf("[DEBUG] Processing public key from %s", user.Username)
		err := s.handlePublicKey(user, msg)
		s.sendRoomKey(user.Username, user.Conn)
		if err != nil {
			log.Printf("[ERROR] Failed to process public key from %s: %v", user.Username, err)
		}
	}

	if msg.Type == shared.TypePrivate {
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
	targetUsername := msg.To
	msg.Type = shared.TypePrivate
	msg.To = targetUsername

	// Check if the message is from the user themselves
	if msg.From == msg.To {
		log.Printf("[WARN] User %s attempted to send a private message to themselves", msg.From)
		return fmt.Errorf("user %s attempted to message themselves", msg.From)
	}

	if targetUsername == msg.From {
		s.sendErrorToConn(user.Conn, "Cannot send private message to yourself")
		return fmt.Errorf("user %s attempted to message themselves", msg.From)
	}

	log.Printf("[DEBUG] Private message from %s to %s: %s",
		msg.From, targetUsername, msg.Content)

	// Find target user
	targetUser, exists := s.users.GetByUsername(targetUsername)
	log.Printf("[DEBUG] Target raw: %q bytes=%v", targetUsername, []byte(targetUsername))

	for _, u := range s.users.GetAll() {
		log.Printf("[DEBUG] user: username=%q addr=%p", u.Username, u)
	}

	log.Printf("[DEBUG] Lookup for target user %s: exists=%v", targetUsername, exists)
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
	for _, user := range s.users.GetAll() {
		log.Printf("[DEBUG] broadcast: sender=%q | current=%q", msg.From, user.Username)

		if strings.TrimSpace(user.Username) == strings.TrimSpace(msg.From) {
			log.Printf("[DEBUG] Skipping sender %s", user.Username)
			continue
		}

		if err := user.WriteMessage(msg); err != nil {
			log.Printf("[ERROR] Failed to send to %s: %v", user.Username, err)
		}
	}
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

func (s *Server) handlePublicKey(user *shared.User, msg *shared.Message) error {
	log.Printf("[DEBUG] Received public key from %s", user.Username)
	pemData := []byte(msg.Content)
	pubKey, err := shared.ParsePublicKeyFromPEM(pemData)
	if err != nil {
		return fmt.Errorf("invalid public key from %s: %v", user.Username, err)
	}
	s.users.SetPublicKey(user.Username, pubKey)
	log.Printf("[INFO] Stored public key for user %s", user.Username)
	return nil
}

func (s *Server) sendRoomKey(username string, conn net.Conn) {
	user, exists := s.users.GetByUsername(username)
	if !exists {
		log.Printf("[ERROR] Cannot send room key, user not found: %s", username)
		return
	}
	if user.PublicKey == nil {
		log.Printf("[ERROR] Cannot send room key, public key not set for user: %s", username)
		return
	}

	encKeyB64, err := shared.EncryptRoomKey(user.PublicKey, s.roomKey)
	fmt.Print("Encrypted room key for user ", username, ": ", encKeyB64, "\n")
	if err != nil {
		log.Printf("[ERROR] Failed to encrypt room key for %s: %v", username, err)
		return
	}

	msg := &shared.Message{
		Type:         shared.TypeRoomKey,
		From:         "server",
		Content:      "Room key distribution",
		EncryptedKey: encKeyB64,
		Timestamp:    time.Now(),
	}

	if err := user.WriteMessage(msg); err != nil {
		log.Printf("[ERROR] Failed to send room key to %s: %v", username, err)
	} else {
		log.Printf("[INFO] Sent room key to %s", username)
	}
}
func (s *Server) handlePublicKeyRequest(msg *shared.Message) error {
	targetUser, exists := s.users.GetByUsername(msg.To)
	if !exists {
		return fmt.Errorf("user %s not found", msg.To)
	}

	if targetUser.PublicKey == nil {
		return fmt.Errorf("user %s has no public key set", msg.To)
	}

	requester, exists := s.users.GetByUsername(msg.From)
	if !exists {
		return fmt.Errorf("requester %s not found", msg.From)
	}

	if requester.PublicKey == nil {
		return fmt.Errorf("requester %s has no public key set", msg.From)
	}

	// Convert target's public key to PEM format string
	pemPub, _ := shared.PublicKeyToPEM(targetUser.PublicKey)

	// Encrypt the PEM public key using the REQUESTER's public key
	encKeyB64, encDataB64, err := shared.Encrypt(string(pemPub), requester.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt public key for %s: %v", msg.From, err)
	}

	// Send encrypted key and data back to requester
	resp := &shared.Message{
		Type:         shared.TypePublicKeyResponse,
		From:         msg.To,     // the user whose key was requested
		To:           msg.From,   // the requester
		EncryptedKey: encKeyB64,  // AES key encrypted with requester’s RSA pubkey
		Content:      encDataB64, // target's PEM public key encrypted with AES
	}

	return requester.WriteMessage(resp)
}
