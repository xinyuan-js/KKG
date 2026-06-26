package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	AccessTokenTTL  = 30 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
)

type Manager struct {
	redis *redis.Client
}

type Pair struct {
	AccessToken  string
	RefreshToken string
}

type Data struct {
	UserID       uint64    `json:"user_id"`
	Role         string    `json:"role"`
	TokenType    string    `json:"token_type"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func NewManager(redisClient *redis.Client) *Manager {
	return &Manager{redis: redisClient}
}

func (m *Manager) IssuePair(ctx context.Context, userID uint64, role string) (*Pair, error) {
	if m == nil || m.redis == nil {
		return nil, errors.New("session redis is not configured")
	}
	if userID == 0 {
		return nil, errors.New("invalid session user")
	}
	accessToken, err := randomToken()
	if err != nil {
		return nil, err
	}
	refreshToken, err := randomToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	accessData := Data{
		UserID:       userID,
		Role:         strings.TrimSpace(role),
		TokenType:    "access",
		RefreshToken: refreshToken,
		IssuedAt:     now,
		ExpiresAt:    now.Add(AccessTokenTTL),
	}
	refreshData := Data{
		UserID:      userID,
		Role:        strings.TrimSpace(role),
		TokenType:   "refresh",
		AccessToken: accessToken,
		IssuedAt:    now,
		ExpiresAt:   now.Add(RefreshTokenTTL),
	}
	if err := m.set(ctx, accessKey(accessToken), accessData, AccessTokenTTL); err != nil {
		return nil, err
	}
	if err := m.set(ctx, refreshKey(refreshToken), refreshData, RefreshTokenTTL); err != nil {
		_ = m.redis.Del(ctx, accessKey(accessToken)).Err()
		return nil, err
	}
	return &Pair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (m *Manager) ValidateAccess(ctx context.Context, token string) (*Data, error) {
	data, err := m.get(ctx, accessKey(token))
	if err != nil {
		return nil, err
	}
	if data.TokenType != "access" || data.UserID == 0 {
		return nil, errors.New("invalid access session")
	}
	return data, nil
}

func (m *Manager) Refresh(ctx context.Context, refreshToken string) (*Pair, *Data, error) {
	data, err := m.get(ctx, refreshKey(refreshToken))
	if err != nil {
		return nil, nil, err
	}
	if data.TokenType != "refresh" || data.UserID == 0 {
		return nil, nil, errors.New("invalid refresh session")
	}
	if err := m.DeletePair(ctx, data.AccessToken, refreshToken); err != nil {
		return nil, nil, err
	}
	pair, err := m.IssuePair(ctx, data.UserID, data.Role)
	if err != nil {
		return nil, nil, err
	}
	return pair, data, nil
}

func (m *Manager) DeletePair(ctx context.Context, accessToken string, refreshToken string) error {
	if m == nil || m.redis == nil {
		return nil
	}
	keys := make([]string, 0, 2)
	if strings.TrimSpace(accessToken) != "" {
		keys = append(keys, accessKey(accessToken))
	}
	if strings.TrimSpace(refreshToken) != "" {
		keys = append(keys, refreshKey(refreshToken))
	}
	if len(keys) == 0 {
		return nil
	}
	return m.redis.Del(ctx, keys...).Err()
}

func (m *Manager) set(ctx context.Context, key string, data Data, ttl time.Duration) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return m.redis.Set(ctx, key, raw, ttl).Err()
}

func (m *Manager) get(ctx context.Context, key string) (*Data, error) {
	if m == nil || m.redis == nil {
		return nil, errors.New("session redis is not configured")
	}
	raw, err := m.redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}
	var data Data
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate session token failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func accessKey(token string) string {
	return "auth:access:" + tokenHash(token)
}

func refreshKey(token string) string {
	return "auth:refresh:" + tokenHash(token)
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
