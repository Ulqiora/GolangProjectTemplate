package transaction_manager_test

import (
	"context"
	"errors"
	"testing"

	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/mocks"
	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
)

func TestCtxManagerUsesTransactionFromContext(t *testing.T) {
	t.Parallel()

	tx := &mocks.Connection{}
	manager := transactionmanager.NewCtxManager(&mocks.Postgres{
		MasterConnectionFunc: func(ctx context.Context) (postgres.Connection, error) {
			t.Fatal("master connection should not be called")
			return nil, nil
		},
	}, &mocks.Logger{})

	got, cleanup, err := manager.GetDefaultOrTx(context.WithValue(context.Background(), transactionmanager.TxKey, tx))
	if err != nil {
		t.Fatalf("GetDefaultOrTx returned error: %v", err)
	}
	cleanup()
	if got != tx {
		t.Fatal("expected transaction executor from context")
	}
}

func TestCtxManagerReturnsConnectionAndCleanup(t *testing.T) {
	t.Parallel()

	conn := &mocks.Connection{}
	manager := transactionmanager.NewCtxManager(&mocks.Postgres{
		MasterConnectionFunc: func(ctx context.Context) (postgres.Connection, error) {
			return conn, nil
		},
	}, &mocks.Logger{})

	got, cleanup, err := manager.GetDefaultOrTx(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultOrTx returned error: %v", err)
	}
	if got != conn {
		t.Fatal("expected acquired connection")
	}
	cleanup()
	if conn.CloseCalls != 1 {
		t.Fatalf("expected connection close once, got %d", conn.CloseCalls)
	}
}

func TestCtxManagerReturnsProviderError(t *testing.T) {
	t.Parallel()

	providerErr := errors.New("provider")
	manager := transactionmanager.NewCtxManager(&mocks.Postgres{
		ReadOnlyConnectionFunc: func(ctx context.Context) (postgres.Connection, error) {
			return nil, providerErr
		},
	}, &mocks.Logger{})

	if _, _, err := manager.GetReadOnlyOrTx(context.Background()); !errors.Is(err, providerErr) {
		t.Fatalf("expected provider error, got %v", err)
	}
}
