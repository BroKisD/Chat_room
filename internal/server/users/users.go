package users

import (
	"chatroom/internal/shared"
	"sync"
)

type Manager struct {
	users map[string]*shared.User
	mu    sync.RWMutex
}

func New() *Manager {
	return &Manager{
		users: make(map[string]*shared.User),
	}
}

func (m *Manager) Add(user *shared.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
}

func (m *Manager) Remove(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, userID)
}

func (m *Manager) Get(userID string) (*shared.User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, exists := m.users[userID]
	return user, exists
}

func (m *Manager) List() []*shared.User {
	m.mu.RLock()
	defer m.mu.RUnlock()
	users := make([]*shared.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}
	return users
}

func (m *Manager) GetAll() []*shared.User {
	return m.List()
}

func (m *Manager) RemoveByUsername(username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, user := range m.users {
		if user.Username == username {
			delete(m.users, id)
			return
		}
	}
}
