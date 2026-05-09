package postgresbase

import (
	"testing"

	"github.com/doug-martin/goqu/v9"
	"github.com/stretchr/testify/require"
)

type testModel struct {
	ID   string
	Name string
}

func (m *testModel) Params() map[string]interface{} {
	return map[string]interface{}{
		"id":   m.ID,
		"name": m.Name,
	}
}

func (m *testModel) Fields() []string {
	return []string{"id", "name"}
}

func (m *testModel) PrimaryKey() (string, any) {
	return "id", m.ID
}

func TestLockOptionsSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lock LockOptions
		want string
	}{
		{
			name: "default",
			lock: LockOptions{},
			want: "FOR UPDATE",
		},
		{
			name: "no key update nowait",
			lock: LockOptions{Mode: LockForNoKeyUpdate, WaitPolicy: LockNoWait},
			want: "FOR NO KEY UPDATE NOWAIT",
		},
		{
			name: "share skip locked of table",
			lock: LockOptions{Mode: LockForShare, WaitPolicy: LockSkipLocked, OfTables: []string{"jobs"}},
			want: "FOR SHARE OF jobs SKIP LOCKED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.lock.SQL()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLockOptionsSQLRejectsUnsafeTableName(t *testing.T) {
	t.Parallel()

	_, err := (LockOptions{OfTables: []string{"jobs;drop"}}).SQL()
	require.Error(t, err)
}

func TestSelectForLockSQL(t *testing.T) {
	t.Parallel()

	repo := RepositoryImpl[*testModel]{
		dialect:   goqu.Dialect("postgres"),
		tableName: "jobs",
	}

	sql, args, err := repo.selectForLockSQL(
		goqu.T("jobs").Col("id").Eq("job-1"),
		LockOptions{Mode: LockForUpdate, WaitPolicy: LockSkipLocked},
		WithLimit(1),
	)

	require.NoError(t, err)
	require.Equal(t, `SELECT * FROM "jobs" WHERE ("jobs"."id" = $1) LIMIT $2 FOR UPDATE SKIP LOCKED`, sql)
	require.Equal(t, []any{"job-1", int64(1)}, args)
}

func TestParamsWithout(t *testing.T) {
	t.Parallel()

	got := paramsWithout(map[string]interface{}{
		"id":         "1",
		"name":       "name",
		"created_at": "now",
	}, "id", "created_at")

	require.Equal(t, map[string]interface{}{"name": "name"}, got)
}

func TestPrimaryKeyColumnSupportsPointerModel(t *testing.T) {
	t.Parallel()

	repo := RepositoryImpl[*testModel]{}
	require.Equal(t, "id", repo.primaryKeyColumn())
}
