package shared

import (
	"net"
	"sync"
	"time"
)

type MessageType string

const (
	TypeAuth         MessageType = "auth"      // Authentication messages
	TypeAuthResponse MessageType = "auth_resp" // Authentication response
	TypePublic       MessageType = "public"    // Public messages
	TypePrivate      MessageType = "private"   // Private messages
	TypeUserList     MessageType = "user_list" // User list updates
	TypeError        MessageType = "error"     // Error messages
	TypeJoin         MessageType = "join"      // User join notification
	TypeLeave        MessageType = "leave"     // User leave notification
)

type Message struct {
	Type      MessageType `json:"type"`
	From      string      `json:"from,omitempty"`
	To        string      `json:"to,omitempty"`
	Content   string      `json:"content"`
	Timestamp time.Time   `json:"timestamp"`
	Users     []string    `json:"users,omitempty"`   // For user list updates
	Success   bool        `json:"success,omitempty"` // For auth responses
	Error     string      `json:"error,omitempty"`   // For error messages
}

type User struct {
	Username string    `json:"username"`
	JoinedAt time.Time `json:"joinedAt"`
	Conn     net.Conn  `json:"-"`
	writeMu  sync.Mutex
}

func (u *User) WriteMessage(msg *Message) error {
	u.writeMu.Lock()
	defer u.writeMu.Unlock()
	return WriteMessage(u.Conn, msg)
}
