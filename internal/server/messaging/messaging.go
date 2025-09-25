package messaging

import (
	"sync"

	"chatroom/internal/shared"
)

type Messenger struct {
	mu sync.RWMutex
}

func New() *Messenger {
	return &Messenger{}
}

func (m *Messenger) Broadcast(msg *shared.Message) error {
	// Broadcasting implementation
	return nil
}

func (m *Messenger) SendPrivate(msg *shared.Message) error {
	// Private messaging implementation
	return nil
}
