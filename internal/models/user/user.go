package user

type User struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	SerialPassport string `json:"serial_passport"`
	NumberPassport string `json:"number_passport"`
}
