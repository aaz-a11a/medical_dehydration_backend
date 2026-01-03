package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionData хранит данные сессии
type SessionData struct {
	UserID      uint   `json:"user_id"`
	Login       string `json:"login"`
	IsModerator bool   `json:"is_moderator"`
}

// SessionService управляет сессиями в Redis
type SessionService struct {
	client *redis.Client
	ttl    time.Duration
}

// NewSessionService создает новый сервис сессий
func NewSessionService(host string, port int, password string, db int) (*SessionService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &SessionService{
		client: client,
		ttl:    24 * time.Hour, // сессия живет 24 часа
	}, nil
}

// Create создает новую сессию
func (s *SessionService) Create(ctx context.Context, sessionID string, data SessionData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, "session:"+sessionID, jsonData, s.ttl).Err()
}

// Get получает данные сессии
func (s *SessionService) Get(ctx context.Context, sessionID string) (*SessionData, error) {
	val, err := s.client.Get(ctx, "session:"+sessionID).Result()
	if err == redis.Nil {
		return nil, nil // сессия не найдена
	}
	if err != nil {
		return nil, err
	}

	var data SessionData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}

	return &data, nil
}

// Delete удаляет сессию
func (s *SessionService) Delete(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, "session:"+sessionID).Err()
}

// Extend продлевает время жизни сессии
func (s *SessionService) Extend(ctx context.Context, sessionID string) error {
	return s.client.Expire(ctx, "session:"+sessionID, s.ttl).Err()
}

// Close закрывает соединение с Redis
func (s *SessionService) Close() error {
	return s.client.Close()
}
