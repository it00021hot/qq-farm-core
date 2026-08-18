package wxlogin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// QuickSession is an in-memory desktop WeChat fast_login session.
type QuickSession struct {
	ID        string
	Owner     string
	CreatedAt time.Time
}

// QuickStore keeps active quick-login sessions.
type QuickStore struct {
	mu       sync.Mutex
	sessions map[string]*QuickSession
	svc      *WxLoginService
}

func NewQuickStore() *QuickStore {
	return &QuickStore{
		sessions: make(map[string]*QuickSession),
		svc:      NewWxLoginService(),
	}
}

func (s *QuickStore) Create(owner string) (*QuickSession, error) {
	id, err := randomQuickSessionID()
	if err != nil {
		return nil, err
	}
	session := &QuickSession{
		ID:        id,
		Owner:     owner,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.pruneLocked(time.Now())
	s.sessions[id] = session
	s.mu.Unlock()
	return session, nil
}

func (s *QuickStore) Take(owner, sessionID string) (*QuickSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	session := s.sessions[sessionID]
	if session == nil || session.Owner != owner {
		return nil, fmt.Errorf("Quick login session not found or expired")
	}
	delete(s.sessions, sessionID)
	return session, nil
}

func (s *QuickStore) Peek(owner, sessionID string) (*QuickSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(time.Now())
	session := s.sessions[sessionID]
	if session == nil || session.Owner != owner {
		return nil, fmt.Errorf("Quick login session not found or expired")
	}
	return session, nil
}

func (s *QuickStore) Detect(ctx context.Context, owner, sessionID string) (*LocalWechatProfile, error) {
	if _, err := s.Peek(owner, sessionID); err != nil {
		return nil, err
	}
	return s.svc.DetectDesktopWechat(ctx)
}

func (s *QuickStore) Authorize(ctx context.Context, owner, sessionID string, port uint16, authorizeUUID string, x, y int) (string, error) {
	if _, err := s.Peek(owner, sessionID); err != nil {
		return "", err
	}
	return s.svc.AuthorizeDesktopWechat(ctx, port, authorizeUUID, x, y)
}

func (s *QuickStore) Confirm(owner, sessionID, redirectURL string) (string, YybCredentials, error) {
	if _, err := s.Take(owner, sessionID); err != nil {
		return "", YybCredentials{}, err
	}
	oauthCode, err := ParseQuickRedirectURL(redirectURL)
	if err != nil {
		return "", YybCredentials{}, err
	}
	creds, err := s.svc.ExchangeOAuthCode(context.Background(), oauthCode)
	if err != nil {
		return "", YybCredentials{}, err
	}
	code, updated, err := s.svc.MintGatewayCode(context.Background(), creds, TargetMiniProgramID)
	if err != nil {
		return "", YybCredentials{}, err
	}
	return code, updated, nil
}

func (s *QuickStore) PublicView(session *QuickSession) map[string]any {
	return map[string]any{
		"session_id":   session.ID,
		"appid":        OAUTHAppID,
		"scope":        OAuthScope,
		"redirect_uri": OAuthRedirectURI,
		"state":        OAuthState,
		"ports":        DesktopWechatPorts,
		"expires_at":   session.CreatedAt.Add(TaskTTL).Unix(),
	}
}

func (s *QuickStore) pruneLocked(now time.Time) {
	for id, session := range s.sessions {
		if now.Sub(session.CreatedAt) > TaskTTL {
			delete(s.sessions, id)
		}
	}
}

func randomQuickSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
