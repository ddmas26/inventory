package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SessionClaims holds the user data stored in a Redis session.
type SessionClaims struct {
	UserID      string   `json:"user_id"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// SessionStore manages access sessions and refresh tokens in Redis.
type SessionStore struct {
	client     *redis.Client
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// RefreshRecord is the value stored under a refresh token. It links the refresh
// token back to the user and to the access token it was issued alongside, so a
// rotation can revoke both.
type RefreshRecord struct {
	UserID       string `json:"user_id"`
	SessionToken string `json:"session_token"`
}

// NewSessionStore creates a new Redis-backed token store.
// accessTTL is how long an access token stays valid; refreshTTL is how long a
// refresh token can be exchanged for a new token pair.
func NewSessionStore(addr, password string, accessTTL, refreshTTL time.Duration) *SessionStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	return &SessionStore{client: rdb, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// AccessTTL returns how long an access token is valid for.
func (s *SessionStore) AccessTTL() time.Duration {
	return s.accessTTL
}

// Close closes the Redis connection.
func (s *SessionStore) Close() error {
	return s.client.Close()
}

// CreateSession stores claims in Redis and returns a session token.
func (s *SessionStore) CreateSession(ctx context.Context, claims *SessionClaims) (string, error) {
	token := uuid.New().String()
	key := fmt.Sprintf("session:%s", token)

	data, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	if err := s.client.Set(ctx, key, data, s.accessTTL).Err(); err != nil {
		return "", fmt.Errorf("redis set: %w", err)
	}

	return token, nil
}

// GetSession retrieves claims for a session token.
func (s *SessionStore) GetSession(ctx context.Context, token string) (*SessionClaims, error) {
	key := fmt.Sprintf("session:%s", token)

	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // session not found or expired
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var claims SessionClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	return &claims, nil
}

// RefreshSession resets the TTL on an existing session.
func (s *SessionStore) RefreshSession(ctx context.Context, token string) error {
	key := fmt.Sprintf("session:%s", token)
	return s.client.Expire(ctx, key, s.accessTTL).Err()
}

// DeleteSession removes a session from Redis.
func (s *SessionStore) DeleteSession(ctx context.Context, token string) error {
	key := fmt.Sprintf("session:%s", token)
	return s.client.Del(ctx, key).Err()
}

// CreateRefreshToken stores a refresh token bound to a user and the access token
// it was issued with, and returns the token itself.
func (s *SessionStore) CreateRefreshToken(ctx context.Context, userID, sessionToken string) (string, error) {
	token := uuid.New().String()
	key := fmt.Sprintf("refresh:%s", token)

	data, err := json.Marshal(RefreshRecord{UserID: userID, SessionToken: sessionToken})
	if err != nil {
		return "", fmt.Errorf("marshal refresh record: %w", err)
	}

	if err := s.client.Set(ctx, key, data, s.refreshTTL).Err(); err != nil {
		return "", fmt.Errorf("redis set: %w", err)
	}

	return token, nil
}

// GetRefreshToken retrieves the record for a refresh token.
// It returns (nil, nil) when the token is unknown or has expired.
func (s *SessionStore) GetRefreshToken(ctx context.Context, token string) (*RefreshRecord, error) {
	key := fmt.Sprintf("refresh:%s", token)

	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var record RefreshRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("unmarshal refresh record: %w", err)
	}

	return &record, nil
}

// DeleteRefreshToken revokes a refresh token.
func (s *SessionStore) DeleteRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("refresh:%s", token)
	return s.client.Del(ctx, key).Err()
}
