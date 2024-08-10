package usecase

import (
	"context"

	open_telemetry "GolangTemplateProject/pkg/open-telemetry"
)

type UserUsecaseImpl struct {
}

func (u *UserUsecaseImpl) Login(ctx context.Context, username string, password string) (string, error) {
	//ctx.Value()
	ctx, span := open_telemetry.Tracer.Start(ctx, "UserUsecase.Login")
	defer span.End()
	return "", nil
}

type UserUsecase struct {
	UserUsecaseImpl
}

func (u *UserUsecase) Login(ctx context.Context, username string, password string) (string, error) {
	ctx, span := open_telemetry.Tracer.Start(ctx, "UserUsecase.Login")
	defer span.End()

	return u.UserUsecaseImpl.Login(ctx, username, password)

}
