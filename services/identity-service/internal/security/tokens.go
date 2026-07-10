package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	UserID    string   `json:"sub"`
	TenantID  string   `json:"tenantId"`
	AccountID string   `json:"account_id,omitempty"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Scopes    []string `json:"scopes"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret     []byte
	privateKey *rsa.PrivateKey
	kid        string
	issuer     string
	audience   string
	ttl        time.Duration
}

func NewTokenManager(secret []byte, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: secret, ttl: ttl, issuer: "tradeops-identity-service", audience: "tradeops-api", kid: "legacy-hs256"}
}

func NewOIDCTokenManager(privateKey *rsa.PrivateKey, kid, issuer, audience string, ttl time.Duration) *TokenManager {
	return &TokenManager{privateKey: privateKey, kid: kid, issuer: issuer, audience: audience, ttl: ttl}
}

func (m *TokenManager) AccessTokenTTL() time.Duration {
	return m.ttl
}

func (m *TokenManager) CreateAccessToken(userID, tenantID, email string, roles []string) (string, error) {
	now := time.Now().UTC()
	if tenantID == "" {
		tenantID = "default-tenant"
	}
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   userID,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	claims.Scopes = scopesForRoles(roles)
	if m.privateKey != nil {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = m.kid
		return token.SignedString(m.privateKey)
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

func (m *TokenManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if m.privateKey != nil {
			if token.Method != jwt.SigningMethodRS256 {
				return nil, errors.New("unexpected signing method")
			}
			return &m.privateKey.PublicKey, nil
		}
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithAudience(m.audience))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (m *TokenManager) JWKS() map[string]any {
	if m.privateKey == nil {
		return map[string]any{"keys": []any{}}
	}
	return map[string]any{"keys": []any{PublicJWK(&m.privateKey.PublicKey, m.kid)}}
}

func PublicJWK(key *rsa.PublicKey, kid string) map[string]string {
	return map[string]string{
		"kty": "RSA",
		"use": "sig",
		"kid": kid,
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigEndianExponent(key.E)),
	}
}

func LoadOrGenerateRSAKey(path string, logger *slog.Logger) (*rsa.PrivateKey, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, errors.New("invalid PEM private key")
		}
		if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			if rsaKey, ok := key.(*rsa.PrivateKey); ok {
				return rsaKey, nil
			}
		}
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	if logger != nil {
		logger.Warn("IDENTITY_JWT_PRIVATE_KEY_PATH not set; generated ephemeral local OIDC signing key")
	}
	return rsa.GenerateKey(rand.Reader, 2048)
}

func scopesForRoles(roles []string) []string {
	scopes := map[string]struct{}{"orders:read": {}, "portfolio:read": {}}
	for _, role := range roles {
		switch role {
		case "admin", "trading_admin":
			for _, scope := range []string{"orders:write", "orders:cancel", "audit:read", "admin:read", "metrics:read"} {
				scopes[scope] = struct{}{}
			}
		case "trader":
			scopes["orders:write"] = struct{}{}
			scopes["orders:cancel"] = struct{}{}
		case "risk_manager":
			scopes["risk:read"] = struct{}{}
			scopes["admin:read"] = struct{}{}
		}
	}
	out := make([]string, 0, len(scopes))
	for scope := range scopes {
		out = append(out, scope)
	}
	return out
}

func bigEndianExponent(exponent int) []byte {
	if exponent == 0 {
		return []byte{0}
	}
	var bytes []byte
	for exponent > 0 {
		bytes = append([]byte{byte(exponent & 0xff)}, bytes...)
		exponent >>= 8
	}
	return bytes
}

func NewRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashRefreshToken(token string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}
