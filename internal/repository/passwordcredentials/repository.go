package passwordcredentials

import (
	"context"
	"fmt"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/internal/repository/postgresbase"
	"GolangTemplateProject/pkg/adapters/postgres"
	"github.com/google/uuid"
)

type Repository struct {
	base postgresbase.Repository[domain.PasswordCredential]
}

func New(pool postgres.IPostgres) ports.PasswordCredentialRepository {
	return &Repository{
		base: postgresbase.NewRepository[domain.PasswordCredential](pool, domain.PasswordCredentialsTable),
	}
}

func (r *Repository) GetByUserID(ctx context.Context, userID string) (*domain.PasswordCredential, error) {
	parsedID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	credential, err := r.base.FindByPrimaryKey(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (r *Repository) Create(ctx context.Context, credential *domain.PasswordCredential) error {
	return r.base.Create(ctx, *credential)
}
