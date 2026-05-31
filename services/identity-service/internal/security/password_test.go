package security

import "testing"

func TestPasswordHashing(t *testing.T) {
	password := "Password@123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == password {
		t.Fatal("password hash must not equal plain text password")
	}
	if !VerifyPassword(password, hash) {
		t.Fatal("expected password verification to succeed")
	}
	if VerifyPassword("wrong-password", hash) {
		t.Fatal("expected wrong password verification to fail")
	}
}
