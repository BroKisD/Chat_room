package auth

import (
	"net"
	"sync"

	"chatroom/internal/shared"
)

type Authenticator struct {
	sessions map[string]*shared.User
	mu       sync.RWMutex
}

func New() *Authenticator {
	return &Authenticator{
		sessions: make(map[string]*shared.User),
	}
}

func (a *Authenticator) Authenticate(conn net.Conn) (*shared.User, error) {
	// Authentication implementation
	return nil, nil
}
