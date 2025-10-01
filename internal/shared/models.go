package shared

import "net"

type MessageType string

const (
	TextMessage      MessageType = "text"
	FileMessage      MessageType = "file"
	SystemMessage    MessageType = "system"
	MessageTypeChat  MessageType = "chat"
	MessageTypeJoin  MessageType = "join"
	MessageTypeLeave MessageType = "leave"
)

type Message struct {
	Type    MessageType   `json:"type"`
	From    string        `json:"from,omitempty"`
	To      string        `json:"to,omitempty"`
	Content string        `json:"content"`
	File    *FileMetadata `json:"file,omitempty"`
	Sender  string        `json:"sender,omitempty"`
}

type FileMetadata struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
}

type User struct {
	ID       string
	Username string
	Conn     net.Conn
}
