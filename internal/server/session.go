package server

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

const (
	sessionCookieName = "minisnap_session"
	sessionTTL        = 24 * time.Hour
)

type session struct {
	expires time.Time
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]session
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]session)}
}

func (s *sessionStore) Create() (string, time.Time) {
	token := newToken()
	expires := time.Now().Add(sessionTTL)

	s.mu.Lock()
	s.sessions[token] = session{expires: expires}
	s.mu.Unlock()

	return token, expires
}

func (s *sessionStore) Validate(token string) bool {
	if token == "" {
		return false
	}

	s.mu.RLock()
	sess, ok := s.sessions[token]
	s.mu.RUnlock()

	if !ok {
		return false
	}
	if time.Now().After(sess.expires) {
		s.Remove(token)
		return false
	}

	return true
}

func (s *sessionStore) Remove(token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func newToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "fallback-token"
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
