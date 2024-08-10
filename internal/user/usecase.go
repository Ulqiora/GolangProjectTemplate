package user

import "context"

type Usecase interface {
	Login(ctx context.Context, username string, password string) (string, error)
}
