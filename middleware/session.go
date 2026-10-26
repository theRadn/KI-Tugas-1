package middleware

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/hkdf"
)

const (
	sessionKeySize = 16
	macKeySize     = 32
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

type HandshakeRequest struct {
	ClientPublicKey string `json:"client_public_key" binding:"required"`
}

type HandshakeResponse struct {
	SessionID       string    `json:"session_id"`
	ServerPublicKey string    `json:"server_public_key"`
	ExpiresAt       time.Time `json:"expires_at"`
}

func (m *SessionManager) CreateSession(
	clientPublicKeyBytes []byte,
) (HandshakeResult, error) {
	curve := ecdh.X25519()

	if len(clientPublicKeyBytes) != 32 {
		return HandshakeResult{}, ErrInvalidClientPublicKey
	}

	clientPublicKey, err := curve.NewPublicKey(clientPublicKeyBytes)
	if err != nil {
		return HandshakeResult{}, ErrInvalidClientPublicKey
	}

	serverPrivateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return HandshakeResult{}, err
	}

	sharedSecret, err := serverPrivateKey.ECDH(clientPublicKey)
	if err != nil {
		return HandshakeResult{}, ErrInvalidClientPublicKey
	}

	sessionID, err := generateSessionID()
	if err != nil {
		return HandshakeResult{}, err
	}

	aesKey, macKey, err := deriveSessionKeys(sharedSecret, sessionID)
	if err != nil {
		return HandshakeResult{}, err
	}

	expiresAt := time.Now().Add(m.ttl)

	session := Session{
		ID:        sessionID,
		AESKey:    aesKey,
		MACKey:    macKey,
		ExpiresAt: expiresAt,
	}

	m.mu.Lock()
	m.removeExpiredLocked()
	m.sessions[sessionID] = &sessionEntry{
		session: session,
		usedIDs: make(map[string]struct{}),
	}
	m.mu.Unlock()

	return HandshakeResult{
		SessionID:       sessionID,
		ServerPublicKey: serverPrivateKey.PublicKey().Bytes(),
		ExpiresAt:       expiresAt,
	}, nil
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

func generateSessionID() (string, error) {
	b := make([]byte, 32)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func deriveSessionKeys(
	sharedSecret []byte,
	sessionID string,
) ([]byte, []byte, error) {
	if len(sharedSecret) == 0 || sessionID == "" {
		return nil, nil, errors.New("invalid key derivation input")
	}

	info := []byte("my-go-api/v3/" + sessionID)
	reader := hkdf.New(sha256.New, sharedSecret, nil, info)

	keyMaterial := make([]byte, sessionKeySize+macKeySize)
	if _, err := io.ReadFull(reader, keyMaterial); err != nil {
		return nil, nil, err
	}

	aesKey := append([]byte(nil), keyMaterial[:sessionKeySize]...)
	macKey := append([]byte(nil), keyMaterial[sessionKeySize:]...)

	return aesKey, macKey, nil
}

var ErrInvalidClientPublicKey = errors.New("invalid client public key")

type HandshakeResult struct {
	SessionID       string
	ServerPublicKey []byte
	ExpiresAt       time.Time
}

// Optional helper if you need to validate JSON manually elsewhere.
func decodeJSON(body io.Reader, dst any) error {
	decoder := json.NewDecoder(body)
	return decoder.Decode(dst)
}
