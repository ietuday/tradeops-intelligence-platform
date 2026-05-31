package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/domain"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/repository"
	"github.com/ietuday/tradeops-intelligence-platform/services/identity-service/internal/security"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidToken = errors.New("invalid token")

type AuditContext struct {
	CorrelationID string
	IPAddress     string
	UserAgent     string
}

type AuthService struct {
	users           *repository.UserRepository
	tokens          *repository.RefreshTokenRepository
	audit           *repository.AuditRepository
	tokenManager    *security.TokenManager
	refreshTokenTTL time.Duration
}

func NewAuthService(users *repository.UserRepository, tokens *repository.RefreshTokenRepository, audit *repository.AuditRepository, tokenManager *security.TokenManager, refreshTokenTTL time.Duration) *AuthService {
	return &AuthService{users: users, tokens: tokens, audit: audit, tokenManager: tokenManager, refreshTokenTTL: refreshTokenTTL}
}

func (s *AuthService) Register(ctx context.Context, email, password, fullName string, auditCtx AuditContext) (domain.User, error) {
	hash, err := security.HashPassword(password)
	if err != nil {
		return domain.User{}, err
	}
	user, err := s.users.Create(ctx, normalizeEmail(email), hash, strings.TrimSpace(fullName), []string{"trader"})
	if err == nil {
		s.audit.Record(ctx, &user.ID, "register", auditCtx.CorrelationID, auditCtx.IPAddress, auditCtx.UserAgent)
	}
	return user, err
}

func (s *AuthService) Login(ctx context.Context, email, password string, auditCtx AuditContext) (domain.TokenPair, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return domain.TokenPair{}, ErrInvalidCredentials
	}
	if !security.VerifyPassword(password, user.PasswordHash) {
		return domain.TokenPair{}, ErrInvalidCredentials
	}
	pair, err := s.issueTokens(ctx, user)
	if err == nil {
		s.audit.Record(ctx, &user.ID, "login", auditCtx.CorrelationID, auditCtx.IPAddress, auditCtx.UserAgent)
	}
	return pair, err
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string, auditCtx AuditContext) (domain.TokenPair, error) {
	refresh, err := s.tokens.FindActive(ctx, refreshToken)
	if err != nil {
		return domain.TokenPair{}, ErrInvalidToken
	}
	user, err := s.users.FindByID(ctx, refresh.UserID)
	if err != nil {
		return domain.TokenPair{}, ErrInvalidToken
	}
	access, err := s.tokenManager.CreateAccessToken(user.ID, user.Email, user.Roles)
	if err != nil {
		return domain.TokenPair{}, err
	}
	s.audit.Record(ctx, &user.ID, "refresh", auditCtx.CorrelationID, auditCtx.IPAddress, auditCtx.UserAgent)
	return domain.TokenPair{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.tokenManager.AccessTokenTTL().Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string, auditCtx AuditContext) error {
	refresh, _ := s.tokens.FindActive(ctx, refreshToken)
	err := s.tokens.Revoke(ctx, refreshToken)
	if refresh.UserID != "" {
		s.audit.Record(ctx, &refresh.UserID, "logout", auditCtx.CorrelationID, auditCtx.IPAddress, auditCtx.UserAgent)
	}
	return err
}

func (s *AuthService) Me(ctx context.Context, userID string) (domain.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *AuthService) issueTokens(ctx context.Context, user domain.User) (domain.TokenPair, error) {
	access, err := s.tokenManager.CreateAccessToken(user.ID, user.Email, user.Roles)
	if err != nil {
		return domain.TokenPair{}, err
	}
	refresh, err := security.NewRefreshToken()
	if err != nil {
		return domain.TokenPair{}, err
	}
	if err := s.tokens.Store(ctx, user.ID, refresh, s.refreshTokenTTL); err != nil {
		return domain.TokenPair{}, err
	}
	return domain.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.tokenManager.AccessTokenTTL().Seconds()),
	}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
