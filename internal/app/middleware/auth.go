package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"dehydrotationlab4/internal/app/pkg/auth"

	"github.com/gin-gonic/gin"
)

const (
	UserIDKey      = "user_id"
	LoginKey       = "login"
	IsModeratorKey = "is_moderator"
)

// AuthService содержит сервисы для аутентификации
type AuthService struct {
	JWT     *auth.JWTService
	Session *auth.SessionService
}

// AuthMiddleware проверяет аутентификацию через JWT или сессии
func AuthMiddleware(authSvc *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пытаемся получить JWT из заголовка Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := authSvc.JWT.Validate(tokenString)
			if err == nil {
				c.Set(UserIDKey, claims.UserID)
				c.Set(LoginKey, claims.Login)
				c.Set(IsModeratorKey, claims.IsModerator)
				c.Next()
				return
			}
		}

		// Пытаемся получить сессию из cookie
		sessionID, err := c.Cookie("session_id")
		if err == nil && sessionID != "" {
			sessionData, err := authSvc.Session.Get(c.Request.Context(), sessionID)
			if err == nil && sessionData != nil {
				c.Set(UserIDKey, sessionData.UserID)
				c.Set(LoginKey, sessionData.Login)
				c.Set(IsModeratorKey, sessionData.IsModerator)
				// Продлеваем сессию при каждом запросе
				_ = authSvc.Session.Extend(c.Request.Context(), sessionID)
				c.Next()
				return
			}
		}

		// Если не авторизован
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		c.Abort()
	}
}

// OptionalAuthMiddleware опционально проверяет аутентификацию (не требует обязательной авторизации)
func OptionalAuthMiddleware(authSvc *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пытаемся получить JWT из заголовка Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := authSvc.JWT.Validate(tokenString)
			if err == nil {
				c.Set(UserIDKey, claims.UserID)
				c.Set(LoginKey, claims.Login)
				c.Set(IsModeratorKey, claims.IsModerator)
				c.Next()
				return
			}
		}

		// Пытаемся получить сессию из cookie
		sessionID, err := c.Cookie("session_id")
		if err == nil && sessionID != "" {
			sessionData, err := authSvc.Session.Get(c.Request.Context(), sessionID)
			if err == nil && sessionData != nil {
				c.Set(UserIDKey, sessionData.UserID)
				c.Set(LoginKey, sessionData.Login)
				c.Set(IsModeratorKey, sessionData.IsModerator)
				_ = authSvc.Session.Extend(c.Request.Context(), sessionID)
			}
		}

		// Продолжаем даже если не авторизован
		c.Next()
	}
}

// RequireModeratorMiddleware проверяет, что пользователь - модератор
func RequireModeratorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isModerator, exists := c.Get(IsModeratorKey)
		if !exists || !isModerator.(bool) {
			// Debug лог для диагностики проблем с ролью
			uid, _ := c.Get(UserIDKey)
			login, _ := c.Get(LoginKey)
			// Используем стандартный лог gin
			c.Set("_moderator_debug", fmt.Sprintf("moderator check failed: uid=%v login=%v isModerator=%v authHeader=%v", uid, login, isModerator, c.GetHeader("Authorization")))
			c.JSON(http.StatusForbidden, gin.H{"error": "moderator access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetCurrentUserID получает ID текущего пользователя из контекста
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}
	return userID.(uint), true
}

// GetCurrentLogin получает логин текущего пользователя из контекста
func GetCurrentLogin(c *gin.Context) (string, bool) {
	login, exists := c.Get(LoginKey)
	if !exists {
		return "", false
	}
	return login.(string), true
}

// IsCurrentUserModerator проверяет, является ли текущий пользователь модератором
func IsCurrentUserModerator(c *gin.Context) bool {
	isModerator, exists := c.Get(IsModeratorKey)
	if !exists {
		return false
	}
	return isModerator.(bool)
}
