package shared

type Message struct {
	Type    MessageType
	From    string
	To      string
	Content string
	File    *FileMetadata
}

type MessageType int

const (
	TextMessage MessageType = iota
	FileMessage
	SystemMessage
)

type FileMetadata struct {
	Name        string
	Size        int64
	ContentType string
}

type User struct {
	ID       string
	Username string
}
