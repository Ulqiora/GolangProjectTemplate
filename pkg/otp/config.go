package otp

import (
	"github.com/pquerna/otp/totp"
)

func GenerateOTPInfo(options totp.GenerateOpts) (string, string, error) {
	otpKey, err := totp.Generate(options)
	if err != nil {
		return "", "", err
	}
	return otpKey.Secret(), otpKey.URL(), nil
}
