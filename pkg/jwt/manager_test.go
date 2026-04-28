package jwt

import (
	"testing"
	"time"
)

func TestJWTManager_GenerateAndVerify(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("secret", time.Hour)
	token, err := manager.Generate("user-1", "user@example.com")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	claims, err := manager.Verify(token)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "user@example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWTManager_VerifyRejectsWrongSecret(t *testing.T) {
	t.Parallel()

	token, err := NewJWTManager("secret", time.Hour).Generate("user-1", "user@example.com")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if _, err := NewJWTManager("other", time.Hour).Verify(token); err == nil {
		t.Fatal("expected verify error")
	}
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	t.Parallel()

	token, err := NewJWTManager("secret", time.Hour).GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}
}
