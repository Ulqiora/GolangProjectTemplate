package transaction_manager_test

import (
	"context"
	"errors"
	"testing"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/mocks"
	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

func TestTransactionManagerReturnsErrorForNilPool(t *testing.T) {
	t.Parallel()

	manager := transactionmanager.NewWithOptions(nil, transactionmanager.WithLogger(&mocks.Logger{}), transactionmanager.WithRegisterer(prometheus.NewRegistry()))
	err := manager.Do(context.Background(), func(ctx context.Context) error {
		t.Fatal("callback should not be called")
		return nil
	})
	if err == nil || err.Error() != "transaction manager pool is nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTransactionManagerRollsBackCallbackError(t *testing.T) {
	t.Parallel()

	callbackErr := errors.New("callback")
	tx := &mocks.Tx{}
	conn := &mocks.Connection{
		BeginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return tx, nil
		},
	}
	manager := transactionmanager.NewWithOptions(&mocks.Postgres{
		MasterConnectionFunc: func(ctx context.Context) (postgres.Connection, error) {
			return conn, nil
		},
	}, transactionmanager.WithLogger(&mocks.Logger{}), transactionmanager.WithRegisterer(prometheus.NewRegistry()))

	err := manager.Do(context.Background(), func(ctx context.Context) error {
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("expected callback error, got %v", err)
	}
	if tx.RollbackCalls != 1 || tx.CommitCalls != 0 {
		t.Fatalf("expected rollback only, got rollback=%d commit=%d", tx.RollbackCalls, tx.CommitCalls)
	}
	if conn.CloseCalls != 1 {
		t.Fatalf("expected connection close once, got %d", conn.CloseCalls)
	}
}
