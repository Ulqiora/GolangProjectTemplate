package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"GolangTemplateProject/internal/domain"
	"GolangTemplateProject/pkg/mocks"
)

func TestWorkerRunOnceMarksMessageSentAfterClaim(t *testing.T) {
	repo := newWorkerOutboxRepo(&domain.OutboxMessage{
		ID:         domain.NewUUIDv7(),
		Topic:      "users.created",
		MessageKey: "user-1",
		Payload:    []byte(`{"user_id":"user-1"}`),
		Status:     domain.OutboxStatusPending,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})
	publisher := &fakePublisher{}
	worker := NewWorker(&mocks.TransactionManager{}, repo, publisher, nil, nil)

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("worker returned error: %v", err)
	}
	if repo.message.Status != domain.OutboxStatusSent {
		t.Fatalf("expected sent status, got %s", repo.message.Status)
	}
	if got := repo.operations; len(got) < 3 || got[0] != "pick" || got[1] != "processing" || got[len(got)-1] != "sent" {
		t.Fatalf("unexpected operation order: %#v", got)
	}
}

func TestWorkerRunOnceSchedulesRetryOnPublishError(t *testing.T) {
	repo := newWorkerOutboxRepo(&domain.OutboxMessage{
		ID:           domain.NewUUIDv7(),
		Topic:        "users.created",
		MessageKey:   "user-1",
		Payload:      []byte(`{"user_id":"user-1"}`),
		Status:       domain.OutboxStatusPending,
		AttemptCount: 1,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})
	publisher := &fakePublisher{err: errors.New("kafka unavailable")}
	worker := NewWorker(&mocks.TransactionManager{}, repo, publisher, nil, nil)

	before := time.Now().UTC()
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("worker returned error: %v", err)
	}
	if repo.message.Status != domain.OutboxStatusFailed {
		t.Fatalf("expected failed status, got %s", repo.message.Status)
	}
	if repo.message.NextAttemptAt == nil || !repo.message.NextAttemptAt.After(before) {
		t.Fatalf("expected next attempt to be scheduled, got %#v", repo.message.NextAttemptAt)
	}
}

type workerOutboxRepo struct {
	message    *domain.OutboxMessage
	operations []string
}

func newWorkerOutboxRepo(message *domain.OutboxMessage) *workerOutboxRepo {
	return &workerOutboxRepo{message: message}
}

func (r *workerOutboxRepo) Create(context.Context, *domain.OutboxMessage) error { return nil }

func (r *workerOutboxRepo) PickPending(context.Context, uint) ([]*domain.OutboxMessage, error) {
	r.operations = append(r.operations, "pick")
	if r.message.Status == domain.OutboxStatusSent {
		return nil, nil
	}
	return []*domain.OutboxMessage{r.message}, nil
}

func (r *workerOutboxRepo) MarkProcessing(_ context.Context, _ string) error {
	r.operations = append(r.operations, "processing")
	r.message.Status = domain.OutboxStatusProcessing
	return nil
}

func (r *workerOutboxRepo) MarkSent(_ context.Context, _ string) error {
	r.operations = append(r.operations, "sent")
	r.message.Status = domain.OutboxStatusSent
	now := time.Now().UTC()
	r.message.PublishedAt = &now
	return nil
}

func (r *workerOutboxRepo) MarkFailed(_ context.Context, _ string, lastError string, nextAttemptAt time.Time) error {
	r.operations = append(r.operations, "failed")
	r.message.Status = domain.OutboxStatusFailed
	r.message.LastError = lastError
	r.message.NextAttemptAt = &nextAttemptAt
	return nil
}

type fakePublisher struct {
	err error
}

func (p *fakePublisher) Publish(context.Context, *domain.OutboxMessage) error {
	return p.err
}
