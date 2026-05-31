package security

import (
	"testing"
	"time"
)

func TestJWTCreationAndValidation(t *testing.T) {
	manager := NewTokenManager([]byte("test-secret-with-enough-length"), 15*time.Minute)
	token, err := manager.CreateAccessToken("user-1", "trader@example.com", []string{"trader"})
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	claims, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken returned error: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "trader@example.com" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "trader" {
		t.Fatalf("unexpected roles: %#v", claims.Roles)
	}
}

func TestJWTRejectsWrongSecret(t *testing.T) {
	manager := NewTokenManager([]byte("test-secret"), 15*time.Minute)
	token, err := manager.CreateAccessToken("user-1", "trader@example.com", []string{"trader"})
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	other := NewTokenManager([]byte("other-secret"), 15*time.Minute)
	if _, err := other.ValidateAccessToken(token); err == nil {
		t.Fatal("expected validation with wrong secret to fail")
	}
}
