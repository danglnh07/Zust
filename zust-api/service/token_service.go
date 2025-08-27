package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	db "zust/db/sqlc"
	"zust/util"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTService struct to hold the configuration for JWT
type JWTService struct {
	SecretKey                  []byte
	TokenExpirationTime        time.Duration
	RefreshTokenExpirationTime time.Duration
}

// JWT custom claims struct
type CustomClaims struct {
	ID                   string `json:"id"`
	Username             string `json:"username"`
	Avatar               string `json:"avatar"`
	Role                 string `json:"role"`
	TokenType            string `json:"token_type"`
	Version              int    `json:"version"`
	jwt.RegisteredClaims        // Embed the JWT Registered claims
}

// Function to create a new JWTService
func NewJWTService() *JWTService {
	// Load configuration from .env
	config := util.GetConfig()

	return &JWTService{
		SecretKey:                  []byte(config.SecretKey),
		TokenExpirationTime:        config.TokenExpirationTime * time.Minute,
		RefreshTokenExpirationTime: config.RefreshTokenExpirationTime * time.Minute,
	}
}

// Method to create a new JWT token. It receive account ID, username, avatar, role, token type (access or refresh),
// version and expiration time then return the signed token (string) or error
func (service *JWTService) CreateToken(
	accID, tokenType string, version int, expiration time.Duration) (string, error) {
	// Check for token type value
	if tokenType = strings.TrimSpace(tokenType); tokenType != "refresh-token" && tokenType != "access-token" {
		return "", fmt.Errorf("invalid token type, only accept refresh-token or access-token")
	}

	// Create custom JWT claim
	claims := CustomClaims{
		ID:        accID,
		TokenType: tokenType,
		Version:   version,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "Zust",                                         // Who issue this token
			Subject:   accID,                                          // Whom the token is about
			IssuedAt:  jwt.NewNumericDate(time.Now()),                 // When the token is created
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)), // When the token is expired
		},
	}

	// Generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenStr, err := token.SignedString(service.SecretKey)
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

// Method to verify the token. It receive the signed token (string) and return the custom claims or error
func (service *JWTService) VerifyToken(signedToken string, query *db.Queries) (*CustomClaims, error) {
	// Use custom parser with deley to 30 secs
	parser := jwt.NewParser(jwt.WithLeeway(30 * time.Second))

	// Parse token
	parsedToken, err := parser.ParseWithClaims(signedToken, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		// Check for signing method to avoid [alg: none] trick
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return service.SecretKey, nil
	})

	// Check if token parsing success
	if err != nil {
		return nil, err
	}

	// Extract claims from token
	claims, ok := parsedToken.Claims.(*CustomClaims)
	if !ok || !parsedToken.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	// Check if this is the correct issuer
	if claims.Issuer != "Zust" {
		return nil, fmt.Errorf("invalid issuer")
	}

	// Check if the token type is correct
	if claims.TokenType != "refresh-token" && claims.TokenType != "access-token" {
		return nil, fmt.Errorf("invalid token type")
	}

	// Check if token version is correct with database
	var uuid uuid.UUID
	err = uuid.Scan(claims.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID in token")
	}
	version, err := query.GetTokenVersion(context.Background(), uuid)
	if err != nil {
		return nil, fmt.Errorf("cannot get token version from database: %v", err)
	}
	if int(version) != claims.Version {
		return nil, fmt.Errorf("token version is not valid")
	}

	return claims, nil
}

// Method to refresh the access token. It receive the refresh token (string) and return a new access token (string)
// or error
func (service *JWTService) RefreshToken(refreshToken string, query *db.Queries) (string, error) {
	// First, check if the refresh token is valid and not expire
	claims, err := service.VerifyToken(refreshToken, query)
	if err != nil {
		return "", err
	}

	// Check if this really the refresh token
	if claims.TokenType != "refresh-token" {
		return "", fmt.Errorf("invalid token type")
	}

	// Create new refresh token
	newToken, err := service.CreateToken(claims.ID, "access-token",
		claims.Version, service.TokenExpirationTime)
	if err != nil {
		return "", err
	}
	return newToken, nil
}
