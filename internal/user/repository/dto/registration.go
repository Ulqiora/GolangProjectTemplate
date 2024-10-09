package dto

type RegistrationUserInfoDTO struct {
	Login         string `db:"login"`
	Email         string `db:"email"`
	Password      string `db:"password"`
	OtpSecret     string `db:"otp_secret"`
	UrlOtpCode    string `db:"url_otp_code"`
	OtpCryptNonce string `db:"nonce"`
}
