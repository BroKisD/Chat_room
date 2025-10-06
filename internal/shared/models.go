package shared

import (
	"crypto/rsa"
	"net"
	"sync"
	"time"
)

type MessageType string

const (
	TypeAuth              MessageType = "auth"      // Authentication messages
	TypeAuthResponse      MessageType = "auth_resp" // Authentication response
	TypePublic            MessageType = "public"    // Public messages
	TypePrivate           MessageType = "private"   // Private messages
	TypeUserList          MessageType = "user_list" // User list updates
	TypeError             MessageType = "error"     // Error messages
	TypeJoin              MessageType = "join"      // User join notification
	TypeLeave             MessageType = "leave"     // User leave notification
	TypePublicKey         MessageType = "public_key"
	TypeRoomKey           MessageType = "room_key"
	TypePublicKeyRequest  MessageType = "public_key_request"
	TypePublicKeyResponse MessageType = "public_key_response"
)

type Message struct {
	Type          MessageType `json:"type"`
	From          string      `json:"from,omitempty"`
	To            string      `json:"to,omitempty"`
	Content       string      `json:"content"`
	Timestamp     time.Time   `json:"timestamp"`
	Users         []string    `json:"users,omitempty"`          // For user list updates
	Success       bool        `json:"success,omitempty"`        // For auth responses
	Error         string      `json:"error,omitempty"`          // For error messages
	EncryptedKey  string      `json:"encrypted_key,omitempty"`  // base64 of RSA-encrypted AES key
	EncryptedData string      `json:"encrypted_data,omitempty"` // base64 of AES-encrypted content
}

type User struct {
	Username     string    `json:"username"`
	JoinedAt     time.Time `json:"joinedAt"`
	Conn         net.Conn  `json:"-"`
	writeMu      sync.Mutex
	PublicKey    *rsa.PublicKey `json:"-"`
	PublicKeyPEM string         `json:"publicKeyPEM"`
}

func (u *User) WriteMessage(msg *Message) error {
	u.writeMu.Lock()
	defer u.writeMu.Unlock()
	return WriteMessage(u.Conn, msg)
}
