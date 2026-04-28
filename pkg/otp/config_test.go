package otp

import (
	"net/url"
	"testing"

	"github.com/pquerna/otp/totp"
)

func TestGenerateOTPInfo(t *testing.T) {
	t.Parallel()

	secret, otpURL, err := GenerateOTPInfo(totp.GenerateOpts{
		Issuer:      "issuer",
		AccountName: "user@example.com",
	})
	if err != nil {
		t.Fatalf("GenerateOTPInfo returned error: %v", err)
	}
	if secret == "" {
		t.Fatal("expected secret")
	}
	parsed, err := url.Parse(otpURL)
	if err != nil {
		t.Fatalf("parse otp url: %v", err)
	}
	if parsed.Scheme != "otpauth" {
		t.Fatalf("expected otpauth url, got %s", parsed.Scheme)
	}
}
