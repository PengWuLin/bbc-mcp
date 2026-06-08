package auth

import (
	"errors"
	"strings"
)

var (
	ErrMissingAuth      = errors.New("缺少认证信息")
	ErrInvalidAuthFormat = errors.New("认证格式错误，应为: Bearer <token>")
	ErrInvalidToken     = errors.New("无效的认证令牌")
)

type AuthMiddleware struct {
	tokens map[string]bool
}

func NewAuthMiddleware(tokens []string) *AuthMiddleware {
	m := make(map[string]bool)
	for _, t := range tokens {
		m[t] = true
	}
	return &AuthMiddleware{tokens: m}
}

func (a *AuthMiddleware) Validate(authHeader string) error {
	if authHeader == "" {
		return ErrMissingAuth
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ErrInvalidAuthFormat
	}
	token := parts[1]
	if !a.tokens[token] {
		return ErrInvalidToken
	}
	return nil
}