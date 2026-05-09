package outboxmessages

import (
	"context"
	"fmt"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/internal/ports"
	"GolangTemplateProject/internal/repository/postgresbase"
	"GolangTemplateProject/pkg/adapters/postgres"
	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
)

type Repository struct {
	base postgresbase.Repository[domain.OutboxMessage]
}

func New(pool postgres.IPostgres) ports.OutboxRepository {
	return &Repository{
		base: postgresbase.NewRepository[domain.OutboxMessage](pool, domain.OutboxMessagesTable),
	}
}

func (r *Repository) Create(ctx context.Context, message *domain.OutboxMessage) error {
	return r.base.Create(ctx, *message)
}

func (r *Repository) PickPending(ctx context.Context, limit uint) ([]*domain.OutboxMessage, error) {
	now := time.Now().UTC()
	staleProcessingBefore := now.Add(-5 * time.Minute)

	messages, err := r.base.SelectForLock(
		ctx,
		goqu.Or(
			goqu.Ex{domain.OutboxMessageStatusField: domain.OutboxStatusPending},
			goqu.And(
				goqu.Ex{domain.OutboxMessageStatusField: domain.OutboxStatusFailed},
				goqu.Or(
					goqu.C(domain.OutboxMessageNextAttemptAtField).IsNull(),
					goqu.C(domain.OutboxMessageNextAttemptAtField).Lte(now),
				),
			),
			goqu.And(
				goqu.Ex{domain.OutboxMessageStatusField: domain.OutboxStatusProcessing},
				goqu.C(domain.OutboxMessageUpdatedAtField).Lte(staleProcessingBefore),
			),
		),
		postgresbase.LockOptions{Mode: postgresbase.LockForUpdate, WaitPolicy: postgresbase.LockSkipLocked},
		postgresbase.WithOrderBy(postgresbase.Asc(domain.OutboxMessageNextAttemptAtField), postgresbase.Asc(domain.OutboxMessageCreatedAtField)),
		postgresbase.WithLimit(limit),
	)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.OutboxMessage, 0, len(messages))
	for i := range messages {
		result = append(result, &messages[i])
	}
	return result, nil
}

func (r *Repository) MarkProcessing(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse outbox id: %w", err)
	}
	now := time.Now().UTC()
	return r.base.UpdateFieldsByPrimaryKey(ctx, parsedID, map[string]interface{}{
		domain.OutboxMessageStatusField:        domain.OutboxStatusProcessing,
		domain.OutboxMessageUpdatedAtField:     now,
		domain.OutboxMessageNextAttemptAtField: nil,
	})
}

func (r *Repository) MarkSent(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse outbox id: %w", err)
	}
	now := time.Now().UTC()
	return r.base.UpdateFieldsByPrimaryKey(ctx, parsedID, map[string]interface{}{
		domain.OutboxMessageStatusField:        domain.OutboxStatusSent,
		domain.OutboxMessagePublishedAtField:   refTime(now),
		domain.OutboxMessageUpdatedAtField:     now,
		domain.OutboxMessageLastErrorField:     "",
		domain.OutboxMessageNextAttemptAtField: nil,
	})
}

func (r *Repository) MarkFailed(ctx context.Context, id string, lastError string, nextAttemptAt time.Time) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("parse outbox id: %w", err)
	}
	now := time.Now().UTC()
	return r.base.UpdateFieldsByPrimaryKey(ctx, parsedID, map[string]interface{}{
		domain.OutboxMessageStatusField:        domain.OutboxStatusFailed,
		domain.OutboxMessageLastErrorField:     lastError,
		domain.OutboxMessageAttemptCountField:  goqu.L(domain.OutboxMessageAttemptCountField + " + 1"),
		domain.OutboxMessageUpdatedAtField:     now,
		domain.OutboxMessageNextAttemptAtField: refTime(nextAttemptAt),
	})
}

func refTime(value time.Time) *time.Time {
	return &value
}
