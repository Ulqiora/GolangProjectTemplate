package mocks

import (
	"testing"

	kafkapkg "GolangTemplateProject/pkg/adapters/kafka"
	"GolangTemplateProject/pkg/adapters/postgres"
	"GolangTemplateProject/pkg/closer"
	"GolangTemplateProject/pkg/cripto/aesgcm"
	"GolangTemplateProject/pkg/cripto/bcrypt"
	"GolangTemplateProject/pkg/logger"
	tracingpkg "GolangTemplateProject/pkg/smart-span/tracing"
	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
)

func TestMocksSatisfyPublicInterfaces(t *testing.T) {
	t.Parallel()

	var _ closer.Closer = (*Closer)(nil)
	var _ aesgcm.Crypter = (*Crypter)(nil)
	var _ bcrypt.Hasher = (*Hasher)(nil)
	var _ logger.Logger = (*Logger)(nil)
	var _ kafkapkg.Runnable = (*Runnable)(nil)
	var _ kafkapkg.TypedProducer[string] = (*TypedProducer[string])(nil)
	var _ kafkapkg.ClosableTypedProducer[string] = (*TypedProducer[string])(nil)
	var _ kafkapkg.TransactionalProducer[string] = (*TransactionalProducer[string])(nil)
	var _ kafkapkg.Consumer = (*Consumer)(nil)
	var _ kafkapkg.ClosableConsumer = (*Consumer)(nil)
	var _ postgres.IPostgres = (*Postgres)(nil)
	var _ postgres.Connection = (*Connection)(nil)
	var _ postgres.SqlExecutor = (*Connection)(nil)
	var _ transactionmanager.TransactionManager = (*TransactionManager)(nil)
	var _ tracingpkg.Tracer = (*Tracer)(nil)
}
