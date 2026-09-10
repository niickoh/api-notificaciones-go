package services

import (
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTService valida tokens ****** con HS256.
type JWTService struct {
	secret []byte
}

// NewJWTService construye un validador JWT.
func NewJWTService(secret string) *JWTService {
	return &JWTService{secret: []byte(secret)}
}

// ValidateBearerHeader valida el header Authorization completo.
func (s *JWTService) ValidateBearerHeader(header string) error {
	if !strings.HasPrefix(header, "Bearer ") {
		return errors.New("authorization header must use bearer token")
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if tokenString == "" {
		return errors.New("empty bearer token")
	}

	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return err
	}

	return nil
}
