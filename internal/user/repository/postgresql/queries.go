package postgresql

const (
	queryUserRegister = `
	INSERT INTO 
	    public.client_registration_info(login, email, hashed_password, otp_secret, otp_secret_url, nonce) 
	VALUES (:login, :email, :password, :otp_secret, :url_otp_code, :nonce) RETURNING id;
`
)
