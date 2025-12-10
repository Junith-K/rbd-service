package services

import (
	"sync"
	"time"
)

type TokenInfo struct {
	UserID    string
	ExpiresAt time.Time
}

// TokenStore manages user sessions
type TokenStore struct {
	tokens map[string]*TokenInfo // token -> TokenInfo
	mu     sync.RWMutex
}

var (
	tokenStore *TokenStore
	once       sync.Once
)

// GetTokenStore returns the singleton token store instance
func GetTokenStore() *TokenStore {
	once.Do(func() {
		tokenStore = &TokenStore{
			tokens: make(map[string]*TokenInfo),
		}
		// Start cleanup goroutine
		go tokenStore.cleanupExpiredTokens()
	})
	return tokenStore
}

// StoreToken stores a token with its associated user ID (30 days expiration)
func (ts *TokenStore) StoreToken(token, userID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tokens[token] = &TokenInfo{
		UserID:    userID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
	}
}

// GetUserID retrieves the user ID associated with a token
func (ts *TokenStore) GetUserID(token string) (string, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	tokenInfo, exists := ts.tokens[token]
	if !exists {
		return "", false
	}
	// Check if token is expired
	if time.Now().After(tokenInfo.ExpiresAt) {
		return "", false
	}
	return tokenInfo.UserID, true
}

// DeleteToken removes a token from the store
func (ts *TokenStore) DeleteToken(token string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.tokens, token)
}

// RefreshToken extends the expiration time of an existing token
func (ts *TokenStore) RefreshToken(token string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tokenInfo, exists := ts.tokens[token]
	if !exists {
		return false
	}
	// Check if token is expired
	if time.Now().After(tokenInfo.ExpiresAt) {
		delete(ts.tokens, token)
		return false
	}
	// Extend expiration by 30 days
	tokenInfo.ExpiresAt = time.Now().Add(30 * 24 * time.Hour)
	return true
}

// cleanupExpiredTokens removes expired tokens periodically
func (ts *TokenStore) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		ts.mu.Lock()
		now := time.Now()
		for token, info := range ts.tokens {
			if now.After(info.ExpiresAt) {
				delete(ts.tokens, token)
			}
		}
		ts.mu.Unlock()
	}
}
