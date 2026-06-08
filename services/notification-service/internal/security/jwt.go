package security

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string   `json:"sub"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

type Validator struct {
	secret []byte
}

func NewValidator(secret []byte) *Validator {
	return &Validator{secret: secret}
}

func (v *Validator) Validate(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return v.secret, nil
	}, jwt.WithIssuer("tradeops-identity-service"))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.UserID == "" {
		claims.UserID = claims.Subject
	}
	if claims.UserID == "" {
		return nil, errors.New("invalid token subject")
	}
	return claims, nil
}
