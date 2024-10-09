package entity

type ID string

type User struct {
	Id             ID     `db:"id"`
	Email          string `db:"email"`
	HashedPassword string `db:"password"`
	OtpCode        string `db:"otp"`
}
