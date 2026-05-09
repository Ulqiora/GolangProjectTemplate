package authusers

import (
	"context"
	"fmt"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/internal/repository/postgresbase"
	"GolangTemplateProject/pkg/adapters/postgres"
	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
)

type Repository struct {
	base postgresbase.Repository[domain.UserAccount]
}

func New(pool postgres.IPostgres) ports.UserAccountRepository {
	return &Repository{
		base: postgresbase.NewRepository[domain.UserAccount](pool, domain.AuthUsersTable),
	}
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.UserAccount, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	account, err := r.base.FindByPrimaryKey(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *Repository) GetByLogin(ctx context.Context, login string) (*domain.UserAccount, error) {
	account, err := r.base.SelectOneBy(ctx, goqu.Ex{domain.AuthUserLoginField: login}, postgresbase.WithLimit(1))
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*domain.UserAccount, error) {
	account, err := r.base.SelectOneBy(ctx, goqu.Ex{domain.AuthUserEmailField: email}, postgresbase.WithLimit(1))
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *Repository) Create(ctx context.Context, account *domain.UserAccount) error {
	return r.base.Create(ctx, *account)
}
