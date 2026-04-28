package bcrypt

import "testing"

func TestBCrypt_HashAndValidate(t *testing.T) {
	t.Parallel()

	hasher := New(Config{Cost: 4})
	hash, err := hasher.Hash("secret")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	if hash == "" || hash == "secret" {
		t.Fatalf("unexpected hash %q", hash)
	}
	if err := hasher.Validate("secret", hash); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if err := hasher.Validate("wrong", hash); err == nil {
		t.Fatal("expected validation error for wrong password")
	}
}
