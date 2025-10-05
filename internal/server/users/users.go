package users

import (
	"crypto/rsa"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"chatroom/internal/shared"
)

type Manager struct {
	users map[string]*shared.User // username -> user
	mu    sync.RWMutex
}

func New() *Manager {
	return &Manager{
		users: make(map[string]*shared.User),
	}
}

// AuthenticateUser tries to authenticate a new user
func (m *Manager) AuthenticateUser(username string, conn net.Conn) (*shared.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Normalize username
	norm := strings.TrimSpace(username)
	norm = strings.TrimPrefix(norm, "@")
	norm = strings.ToLower(norm)

	// Check for duplicate username
	if _, exists := m.users[norm]; exists {
		return nil, fmt.Errorf("username %s is already taken", norm)
	}

	// Create new user
	user := &shared.User{
		Username: norm,
		JoinedAt: time.Now(),
		Conn:     conn,
	}

	// Add to active users
	m.users[norm] = user
	fmt.Print("User authenticated: ", norm, "\n")
	return user, nil
}

func (m *Manager) Remove(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, username)
}

func (m *Manager) GetByUsername(username string) (*shared.User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, exists := m.users[username]
	return user, exists
}

// GetUsernames returns a list of all active usernames
func (m *Manager) GetUsernames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	usernames := make([]string, 0, len(m.users))
	for username := range m.users {
		usernames = append(usernames, username)
	}
	return usernames
}

// GetAll returns all active users
func (m *Manager) GetAll() []*shared.User {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*shared.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users
}

func (m *Manager) SetPublicKey(username string, pubKey *rsa.PublicKey) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if user, exists := m.users[username]; exists {
		user.PublicKey = pubKey
	}
}
func (m *Manager) GetPublicKey(username string) (*rsa.PublicKey, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if user, exists := m.users[username]; exists && user.PublicKey != nil {
		return user.PublicKey, true
	}
	return nil, false
}
