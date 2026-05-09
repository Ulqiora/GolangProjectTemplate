package externalidentities

import (
	"context"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/internal/repository/postgresbase"
	"GolangTemplateProject/pkg/adapters/postgres"
	"github.com/doug-martin/goqu/v9"
)

type Repository struct {
	base postgresbase.Repository[domain.ExternalIdentity]
}

func New(pool postgres.IPostgres) ports.ExternalIdentityRepository {
	return &Repository{
		base: postgresbase.NewRepository[domain.ExternalIdentity](pool, domain.ExternalIdentitiesTable),
	}
}

func (r *Repository) GetByProviderSubject(ctx context.Context, provider domain.AuthProvider, subject string) (*domain.ExternalIdentity, error) {
	identity, err := r.base.SelectOneBy(ctx, goqu.Ex{
		domain.ExternalIdentityProviderField: string(provider),
		domain.ExternalIdentitySubjectField:  subject,
	}, postgresbase.WithLimit(1))
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

func (r *Repository) Create(ctx context.Context, identity *domain.ExternalIdentity) error {
	return r.base.Create(ctx, *identity)
}
