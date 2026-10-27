package middleware

import (
	"sync"
	"time"
)

type Session struct {
	ID        string
	AESKey    []byte
	MACKey    []byte
	ExpiresAt time.Time
}

type sessionEntry struct {
	session Session
	usedIDs map[string]struct{}
}

type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]*sessionEntry
	ttl      time.Duration
}

func NewSessionManager(ttl time.Duration) *SessionManager {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	return &SessionManager{
		sessions: make(map[string]*sessionEntry),
		ttl:      ttl,
	}
}

func (m *SessionManager) GetSession(sessionID string) (Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.sessions[sessionID]
	if !ok {
		return Session{}, false
	}

	if time.Now().After(entry.session.ExpiresAt) {
		delete(m.sessions, sessionID)
		return Session{}, false
	}

	return Session{
		ID:        entry.session.ID,
		AESKey:    append([]byte(nil), entry.session.AESKey...),
		MACKey:    append([]byte(nil), entry.session.MACKey...),
		ExpiresAt: entry.session.ExpiresAt,
	}, true
}

func (m *SessionManager) StoreSession(session Session) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.removeExpiredLocked()
	m.sessions[session.ID] = &sessionEntry{
		session: session,
		usedIDs: make(map[string]struct{}),
	}
}

// ConsumeRequestID rejects a request ID that has already been used
// in this session. Call it only after authenticating the encrypted
// envelope's HMAC, so unauthenticated clients cannot exhaust this map.
func (m *SessionManager) ConsumeRequestID(
	sessionID string,
	requestID string,
) bool {
	if requestID == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.sessions[sessionID]
	if !ok {
		return false
	}

	if time.Now().After(entry.session.ExpiresAt) {
		delete(m.sessions, sessionID)
		return false
	}

	if _, exists := entry.usedIDs[requestID]; exists {
		return false
	}

	entry.usedIDs[requestID] = struct{}{}
	return true
}

func (m *SessionManager) removeExpiredLocked() {
	now := time.Now()

	for id, entry := range m.sessions {
		if now.After(entry.session.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
}
