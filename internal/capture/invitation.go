package capture

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

var (
	ErrInvitationInvalid = errors.New("invitation is invalid")
	ErrInvitationExpired = errors.New("invitation has expired")
	ErrInvitationUsed    = errors.New("invitation has already been used")
)

type Invitation struct {
	RequestID string
	Audience  string
	ExpiresAt time.Time
	UsedAt    time.Time
}

type RedeemedSession struct {
	SessionID string    `json:"session_id"`
	RequestID string    `json:"request_id"`
	Audience  string    `json:"audience"`
	ExpiresAt time.Time `json:"expires_at"`
}

type InvitationService struct {
	mu      sync.Mutex
	now     func() time.Time
	records map[[32]byte]Invitation
}

func NewInvitationService(now func() time.Time) *InvitationService {
	return &InvitationService{now: now, records: make(map[[32]byte]Invitation)}
}

func (s *InvitationService) Issue(requestID, audience string, ttl time.Duration) (string, error) {
	if requestID == "" || audience == "" || ttl <= 0 {
		return "", fmt.Errorf("request, audience and positive ttl are required")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate invitation: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	s.mu.Lock()
	s.records[hash] = Invitation{RequestID: requestID, Audience: audience, ExpiresAt: s.now().UTC().Add(ttl)}
	s.mu.Unlock()
	return token, nil
}

func (s *InvitationService) Redeem(token string) (RedeemedSession, error) {
	hash := sha256.Sum256([]byte(token))
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[hash]
	if !ok {
		return RedeemedSession{}, ErrInvitationInvalid
	}
	if !record.UsedAt.IsZero() {
		return RedeemedSession{}, ErrInvitationUsed
	}
	if !now.Before(record.ExpiresAt) {
		return RedeemedSession{}, ErrInvitationExpired
	}
	record.UsedAt = now
	s.records[hash] = record
	sessionID, err := id.New("cap", 24)
	if err != nil {
		return RedeemedSession{}, err
	}
	return RedeemedSession{SessionID: sessionID, RequestID: record.RequestID, Audience: record.Audience, ExpiresAt: now.Add(15 * time.Minute)}, nil
}
