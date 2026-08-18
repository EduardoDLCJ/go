package middleware

import (
	"fmt"
	"strings"
	"time"

	"apisql/config"
	"apisql/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Config is an alias to the shared application config.
type Config = config.Config

// Custom claims for JWT tokens.
type Claims struct {
	UserID  uint   `json:"user_id"`
	Usuario string `json:"usuario"`
	jwt.RegisteredClaims
}

// TokenType distinguishes between access and refresh tokens.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// GenerateToken creates a signed JWT token.
func GenerateToken(cfg *Config, userID uint, usuario string, tokenType TokenType) (string, error) {
	var expiry time.Duration
	switch tokenType {
	case AccessToken:
		expiry = cfg.JWTAccessExpiry
	case RefreshToken:
		expiry = cfg.JWTRefreshExpiry
	default:
		return "", fmt.Errorf("unknown token type: %s", tokenType)
	}

	claims := Claims{
		UserID:  userID,
		Usuario: usuario,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			Issuer:    "speedmap-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// ParseToken validates and parses a JWT token string.
func ParseToken(cfg *Config, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// AuthRequired is a middleware that requires a valid JWT in the Authorization header.
// It extracts the user_id and sets it in the Gin context.
func AuthRequired(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c)
		if tokenString == "" {
			utils.Unauthorized(c, "Token de acceso requerido")
			c.Abort()
			return
		}

		claims, err := ParseToken(cfg, tokenString)
		if err != nil {
			utils.Unauthorized(c, "Token inválido o expirado")
			c.Abort()
			return
		}

		// Set user info in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("usuario", claims.Usuario)
		c.Next()
	}
}

// OptionalAuth is a middleware that extracts user info from JWT if present,
// but does NOT fail if no token is provided. Useful for endpoints that
// work for both anonymous and authenticated users.
func OptionalAuth(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractBearerToken(c)
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := ParseToken(cfg, tokenString)
		if err != nil {
			// Token invalid but optional — just continue without auth
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("usuario", claims.Usuario)
		c.Next()
	}
}

// extractBearerToken extracts the JWT from the "Authorization: Bearer <token>" header.
func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// GetUserID retrieves the authenticated user's ID from the Gin context.
// Returns 0 if not authenticated.
func GetUserID(c *gin.Context) uint {
	if id, exists := c.Get("user_id"); exists {
		if userID, ok := id.(uint); ok {
			return userID
		}
	}
	return 0
}
