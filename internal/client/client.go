package client

import (
	"chatroom/internal/client/networking"
	"chatroom/internal/shared"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"
)

type Client struct {
	conn              *networking.Connection
	username          string
	activeUsers       []string
	onMessage         func(msg string)
	privateKey        *rsa.PrivateKey
	publicKey         *rsa.PublicKey
	roomKey           []byte
	PublicKeyCache    *PublicKeyCache
	PendingPrivateMsg map[string][]string
	mu                sync.Mutex
}

func New() *Client {
	return &Client{
		conn:              networking.NewConnection(),
		PublicKeyCache:    NewPublicKeyCache(),
		PendingPrivateMsg: make(map[string][]string),
	}
}

func (c *Client) SetMessageHandler(handler func(msg string)) {
	c.onMessage = handler
}

func (c *Client) Connect(address string) error {
	if err := c.conn.Connect(address); err != nil {
		return err
	}

	// Send authentication message
	authMsg := &shared.Message{
		Type:    shared.TypeAuth,
		From:    c.username,
		Content: "auth",
	}
	if err := c.conn.Send(authMsg); err != nil {
		return fmt.Errorf("auth failed: %v", err)
	}

	authResp := <-c.conn.Incoming()
	if authResp.Type != shared.TypeAuthResponse {
		return fmt.Errorf("unexpected response type: %s", authResp.Type)
	}
	if !authResp.Success {
		return fmt.Errorf("authentication failed: %s", authResp.Error)
	}

	priv, pub, err := shared.GenerateRSAKeyPair(2048)
	if err != nil {
		fmt.Printf("failed to generate RSA keys: %v", err)
		return err
	}
	c.privateKey = priv
	c.publicKey = pub

	pemPub, _ := shared.PublicKeyToPEM(pub)
	msg := &shared.Message{
		Type:    shared.TypePublicKey,
		From:    c.username,
		Content: string(pemPub),
	}
	c.conn.Send(msg)

	// Start message listener
	fmt.Print("Public key of client: \n", string(pemPub), "\n")
	go c.handleMessages()

	return nil
}

func (c *Client) Login(username string) error {
	c.username = username
	return nil
}

func (c *Client) SendMessage(content string) error {
	c.displayMessage(fmt.Sprintf("(Global) (You) (%s): %s",
		time.Now().Format("15:04:05"), content))
	fmt.Print("Room key: ", c.roomKey)

	_, encDataB64, err := shared.EncryptWithRoomKey(content, c.roomKey)
	if encDataB64 == "" {
		return fmt.Errorf("encryption failed: empty ciphertext")
	}
	if err != nil {
		return err
	}

	msg := &shared.Message{
		Type:          shared.TypePublic,
		From:          c.username,
		EncryptedData: encDataB64,
		Timestamp:     time.Now(),
	}
	return c.conn.Send(msg)
}

func (c *Client) SendPrivateMessage(target, content string) error {
	target = strings.TrimSpace(target)

	if target == c.username {
		return fmt.Errorf("cannot send private message to yourself")
	}

	targetPubKey, exists := c.PublicKeyCache.Get(target)
	if !exists {

		c.mu.Lock()
		c.PendingPrivateMsg[target] = append(c.PendingPrivateMsg[target], content)
		c.mu.Unlock()

		req := &shared.Message{
			Type: shared.TypePublicKeyRequest,
			From: c.username,
			To:   target,
		}
		return c.conn.Send(req)
	}

	encKeyB64, encDataB64, err := shared.Encrypt(content, targetPubKey)
	if err != nil {
		return err
	}

	msg := &shared.Message{
		Type:         shared.TypePrivate,
		From:         c.username,
		To:           target,
		EncryptedKey: encKeyB64,
		Content:      encDataB64,
		Timestamp:    time.Now(),
	}
	c.displayMessage(fmt.Sprintf("(Private to %s) (You) (%s): %s",
		target, time.Now().Format("15:04:05"), content))

	return c.conn.Send(msg)
}

func (c *Client) Disconnect() error {
	msg := &shared.Message{
		Type:      shared.TypeLeave,
		From:      c.username,
		Timestamp: time.Now(),
	}
	if err := c.conn.Send(msg); err != nil {
		return err
	}
	return c.conn.Close()
}

func (c *Client) GetActiveUsers() []string {
	return c.activeUsers
}

func (c *Client) handleMessages() {
	for msg := range c.conn.Incoming() {

		if msg.From == c.username {
			continue
		}

		switch msg.Type {
		case shared.TypeRoomKey:
			c.handleRoomKey(msg)
		case shared.TypePublic:
			c.formatAndDisplayMessage(msg)
		case shared.TypePrivate:
			msg := c.DecryptPrivateMessage(msg)
			c.formatAndDisplayPrivateMessage(msg)
		case shared.TypeUserList:
			c.activeUsers = msg.Users
			c.notifyUserListUpdate()
		case shared.TypeJoin, shared.TypeLeave:
			c.displaySystemMessage(msg)
		case shared.TypeError:
			c.displayErrorMessage(msg)
		case shared.TypePublicKeyResponse:
			c.handlePublicKeyResponse(msg)
		default:
			fmt.Println("Unknown message type:", msg.Type)
		}
	}
	c.displayMessage("Disconnected from server. Attempting reconnect...")
	go func() {
		for {
			if err := c.ReconnectAndHandshake("127.0.0.1:9000"); err != nil {
				fmt.Println("Reconnect failed:", err)
				time.Sleep(5 * time.Second)
				continue
			} else {
				fmt.Println("Reconnect+handshake success")
				c.displayMessage("Reconnected to server.")
				return
			}
		}
	}()
}

func (c *Client) formatAndDisplayMessage(msg *shared.Message) {
	if msg == nil {
		return
	}
	if strings.TrimSpace(msg.From) == strings.TrimSpace(c.username) {
		return
	}
	msgContent, err := shared.DecryptWithRoomKey(msg.EncryptedData, c.roomKey)
	if err != nil {
		fmt.Println("Failed to decrypt message:", err)
		return
	}
	formatted := fmt.Sprintf("(Global) (%s) %s: %s",
		msg.Timestamp.Format("15:04:05"),
		msg.From,
		msgContent)
	c.displayMessage(formatted)
}

func (c *Client) formatAndDisplayPrivateMessage(msg *shared.Message) {
	if msg == nil {
		return
	}

	if strings.TrimSpace(msg.From) == strings.TrimSpace(c.username) {
		return
	}
	formatted := fmt.Sprintf("(Private) (%s) %s: %s",
		msg.Timestamp.Format("15:04:05"),
		msg.From,
		msg.Content)
	c.displayMessage(formatted)
}

func (c *Client) displaySystemMessage(msg *shared.Message) {
	formatted := fmt.Sprintf("(System) (%s) %s",
		msg.Timestamp.Format("15:04:05"),
		msg.Content)
	c.displayMessage(formatted)
}

func (c *Client) displayErrorMessage(msg *shared.Message) {
	formatted := fmt.Sprintf("(Error) %s", msg.Content)
	c.displayMessage(formatted)
}

func (c *Client) notifyUserListUpdate() {
	formatted := fmt.Sprintf("Active users: %s",
		strings.Join(c.activeUsers, ", "))
	c.displayMessage(formatted)
}

func (c *Client) displayMessage(msg string) {
	if c.onMessage != nil {
		c.onMessage(msg)
	} else {
		fmt.Println(msg)
	}
}

func (c *Client) handleRoomKey(msg *shared.Message) {
	c.roomKey = shared.DecryptRoomKey(msg.EncryptedKey, c.privateKey)
	fmt.Print("User ", c.username, " received room key.\n")
	fmt.Printf("Decrypt using roomKey: %s", base64.StdEncoding.EncodeToString(c.roomKey))
	if c.roomKey == nil {
		fmt.Println("Failed to obtain room key")
		return
	}
	fmt.Print("Decrypted room key: ", c.roomKey, "\n")

	// Here you would typically store the room key for later use
}
func (c *Client) DecryptPrivateMessage(msg *shared.Message) *shared.Message {
	if strings.TrimSpace(msg.From) == strings.TrimSpace(c.username) {
		return nil
	}
	plainBytes, err := shared.Decrypt(msg.EncryptedKey, msg.Content, c.privateKey)
	if err != nil {
		fmt.Println("Failed to decrypt private message:", err)
		return nil
	}
	msg.Content = plainBytes
	return msg
}
func (c *Client) handlePublicKeyResponse(msg *shared.Message) {
	plainPubPEM, err := shared.Decrypt(msg.EncryptedKey, msg.Content, c.privateKey)
	if err != nil {
		fmt.Println("Failed to decode received public key:", err)
		return
	}

	pub, err := shared.ParsePublicKeyFromPEM([]byte(plainPubPEM))
	if err != nil {
		fmt.Println("Failed to parse public key:", err)
		return
	}

	c.PublicKeyCache.Store(msg.From, pub)
	fmt.Printf("Stored public key for %s\n", msg.From)

	c.mu.Lock()
	if pending, ok := c.PendingPrivateMsg[msg.From]; ok {
		for _, content := range pending {
			_ = c.SendPrivateMessage(msg.From, content)
		}
		delete(c.PendingPrivateMsg, msg.From)
	}
	c.mu.Unlock()
}

func (c *Client) ReconnectAndHandshake(address string) error {
	if c.conn != nil {
		_ = c.conn.Close()
	}

	c.conn = networking.NewConnection()

	if err := c.conn.Connect(address); err != nil {
		return err
	}

	authMsg := &shared.Message{
		Type:    shared.TypeAuth,
		From:    c.username,
		Content: "auth",
	}
	if err := c.conn.Send(authMsg); err != nil {
		return fmt.Errorf("auth send failed: %w", err)
	}

	authResp, ok := <-c.conn.Incoming()
	if !ok {
		return fmt.Errorf("connection closed while waiting auth response")
	}
	if authResp.Type != shared.TypeAuthResponse || !authResp.Success {
		return fmt.Errorf("authentication failed: %s", authResp.Error)
	}

	pemPub, _ := shared.PublicKeyToPEM(c.publicKey)
	pubMsg := &shared.Message{
		Type:    shared.TypePublicKey,
		From:    c.username,
		Content: string(pemPub),
	}
	_ = c.conn.Send(pubMsg)

	go c.handleMessages()

	_ = c.conn.Send(&shared.Message{Type: shared.TypeReconnect, From: c.username})

	return nil
}
