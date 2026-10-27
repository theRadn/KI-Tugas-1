package service

import (
	"errors"
	"time"

	"github.com/theRadn/KI-Tugas-1/middleware"
	"github.com/theRadn/KI-Tugas-1/utils"
)

var ErrInvalidClientPublicKey = errors.New("invalid client public key")

type HandshakeResult struct {
	SessionID       string
	ServerPublicKey []byte
	ExpiresAt       time.Time
}

type HandshakeService struct {
	sessionManager *middleware.SessionManager
	ttl            time.Duration
}

func NewHandshakeService(
	sessionManager *middleware.SessionManager,
	ttl time.Duration,
) *HandshakeService {
	return &HandshakeService{
		sessionManager: sessionManager,
		ttl:            ttl,
	}
}

func (s *HandshakeService) CreateSession(
	clientPublicKey []byte,
) (HandshakeResult, error) {
	if len(clientPublicKey) != 32 {
		return HandshakeResult{}, ErrInvalidClientPublicKey
	}

	serverPrivateKey, serverPublicKey, err := utils.GenerateEphemeralX25519()
	if err != nil {
		return HandshakeResult{}, err
	}

	sharedSecret, err := utils.ComputeX25519SharedSecret(
		serverPrivateKey,
		clientPublicKey,
	)
	if err != nil {
		return HandshakeResult{}, ErrInvalidClientPublicKey
	}

	sessionID, err := utils.GenerateSessionID()
	if err != nil {
		return HandshakeResult{}, err
	}

	salt := []byte(sessionID)
	info := []byte("my-go-api/v3")

	keys, err := utils.DeriveSessionKeys(sharedSecret, salt, info)
	if err != nil {
		return HandshakeResult{}, err
	}
	expiresAt := time.Now().Add(s.ttl)

	s.sessionManager.StoreSession(middleware.Session{
		ID:        sessionID,
		AESKey:    keys.AESKey,
		MACKey:    keys.MACKey,
		ExpiresAt: expiresAt,
	})

	return HandshakeResult{
		SessionID:       sessionID,
		ServerPublicKey: serverPublicKey,
		ExpiresAt:       expiresAt,
	}, nil
}
