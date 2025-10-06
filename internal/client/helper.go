package client

import (
	"crypto/rsa"
	"sync"
)

type PublicKeyCache struct {
	cache map[string]*rsa.PublicKey
	mu    sync.RWMutex
}

func NewPublicKeyCache() *PublicKeyCache {
	return &PublicKeyCache{
		cache: make(map[string]*rsa.PublicKey),
	}
}

func (c *PublicKeyCache) Store(username string, pub *rsa.PublicKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[username] = pub
}

func (c *PublicKeyCache) Get(username string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key, ok := c.cache[username]
	return key, ok
}
