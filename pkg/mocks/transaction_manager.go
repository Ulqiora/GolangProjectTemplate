package mocks

import (
	"context"

	transactionmanager "GolangTemplateProject/pkg/transaction-manager"
	"github.com/jackc/pgx/v5"
)

type TransactionManager struct {
	DoFunc  func(context.Context, func(context.Context) error) error
	DoxFunc func(context.Context, func(context.Context) error, pgx.TxOptions) error

	DoCalls  int
	DoxCalls int
}

func (m *TransactionManager) Do(ctx context.Context, fn func(context.Context) error) error {
	m.DoCalls++
	if m.DoFunc != nil {
		return m.DoFunc(ctx, fn)
	}
	return fn(ctx)
}

func (m *TransactionManager) Dox(ctx context.Context, fn func(context.Context) error, opts pgx.TxOptions) error {
	m.DoxCalls++
	if m.DoxFunc != nil {
		return m.DoxFunc(ctx, fn, opts)
	}
	return fn(ctx)
}

var _ transactionmanager.TransactionManager = (*TransactionManager)(nil)
