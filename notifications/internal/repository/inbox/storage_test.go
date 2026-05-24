package inbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestInboxRepository_AddMessage(t *testing.T) {
	t.Parallel()

	db := &fakeDB{}
	repo := NewInboxRepository(db)

	err := repo.AddMessage(context.Background(), "777-paid", []byte(`{"ok":true}`), "topic", 2, 15)

	require.NoError(t, err)
	require.Len(t, db.execCalls, 1)
	require.Contains(t, db.execCalls[0].query, "INSERT INTO notifications.inbox")
	require.Equal(t, []any{"777-paid", []byte(`{"ok":true}`), "topic", int32(2), int64(15)}, db.execCalls[0].args)
}

func TestInboxRepository_AddMessageError(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	repo := NewInboxRepository(&fakeDB{execErr: dbErr})

	err := repo.AddMessage(context.Background(), "777-paid", []byte(`{}`), "topic", 2, 15)

	require.ErrorIs(t, err, dbErr)
}

func TestInboxRepository_AddDeadMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		messageErr error
		wantError  string
	}{
		{name: "with error", messageErr: errors.New("bad json"), wantError: "bad json"},
		{name: "nil error", messageErr: nil, wantError: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := &fakeDB{}
			repo := NewInboxRepository(db)

			err := repo.AddDeadMessage(context.Background(), "dead-key", []byte(`{`), "topic", 1, 9, tt.messageErr)

			require.NoError(t, err)
			require.Len(t, db.execCalls, 1)
			require.Contains(t, db.execCalls[0].query, "DEAD")
			require.Equal(t, []any{"dead-key", []byte(`{`), "topic", int32(1), int64(9), tt.wantError}, db.execCalls[0].args)
		})
	}
}

func TestInboxRepository_AddDeadMessageError(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	repo := NewInboxRepository(&fakeDB{execErr: dbErr})

	err := repo.AddDeadMessage(context.Background(), "dead-key", []byte(`{`), "topic", 1, 9, errors.New("bad json"))

	require.ErrorIs(t, err, dbErr)
}

func TestInboxRepository_GetMessages(t *testing.T) {
	t.Parallel()

	db := &fakeDB{
		rows: &fakeRows{
			items: []Data{
				{IdempotencyKey: "1-paid", Data: []byte(`{"order_id":1}`)},
				{IdempotencyKey: "2-cancelled", Data: []byte(`{"order_id":2}`)},
			},
		},
	}
	repo := NewInboxRepository(db)

	got, err := repo.GetMessages(context.Background(), 10, 30*time.Second, 5)

	require.NoError(t, err)
	require.Equal(t, []Data{
		{IdempotencyKey: "1-paid", Data: []byte(`{"order_id":1}`)},
		{IdempotencyKey: "2-cancelled", Data: []byte(`{"order_id":2}`)},
	}, got)

	require.Len(t, db.queryCalls, 1)
	require.Contains(t, db.queryCalls[0].query, "FOR UPDATE SKIP LOCKED")
	require.Equal(t, int32(5), db.queryCalls[0].args[0])
	require.Equal(t, pgtype.Interval{Microseconds: 30 * int64(time.Second/time.Microsecond), Valid: true}, db.queryCalls[0].args[1])
	require.Equal(t, int32(10), db.queryCalls[0].args[2])
}

func TestInboxRepository_GetMessagesQueryError(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("query error")
	repo := NewInboxRepository(&fakeDB{queryErr: dbErr})

	got, err := repo.GetMessages(context.Background(), 10, time.Minute, 5)

	require.Nil(t, got)
	require.ErrorIs(t, err, dbErr)
}

func TestInboxRepository_GetMessagesRowsError(t *testing.T) {
	t.Parallel()

	rowsErr := errors.New("rows error")
	repo := NewInboxRepository(&fakeDB{
		rows: &fakeRows{err: rowsErr},
	})

	got, err := repo.GetMessages(context.Background(), 10, time.Minute, 5)

	require.Nil(t, got)
	require.ErrorIs(t, err, rowsErr)
}

func TestInboxRepository_MarkAsSuccess(t *testing.T) {
	t.Parallel()

	db := &fakeDB{}
	repo := NewInboxRepository(db)

	err := repo.MarkAsSuccess(context.Background(), []string{"1-paid", "2-cancelled"})

	require.NoError(t, err)
	require.Len(t, db.execCalls, 1)
	require.Contains(t, db.execCalls[0].query, "SUCCESS")
	require.Equal(t, []any{[]string{"1-paid", "2-cancelled"}}, db.execCalls[0].args)
}

func TestInboxRepository_MarkAsSuccessEmptyKeys(t *testing.T) {
	t.Parallel()

	db := &fakeDB{}
	repo := NewInboxRepository(db)

	err := repo.MarkAsSuccess(context.Background(), nil)

	require.NoError(t, err)
	require.Empty(t, db.execCalls)
}

func TestInboxRepository_MarkAsSuccessError(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	repo := NewInboxRepository(&fakeDB{execErr: dbErr})

	err := repo.MarkAsSuccess(context.Background(), []string{"1-paid"})

	require.ErrorIs(t, err, dbErr)
}

func TestInboxRepository_MarkAsFailed(t *testing.T) {
	t.Parallel()

	db := &fakeDB{}
	repo := NewInboxRepository(db)

	err := repo.MarkAsFailed(
		context.Background(),
		[]string{"1-paid", "2-cancelled"},
		[]error{errors.New("first error"), nil},
		5,
		2*time.Second,
	)

	require.NoError(t, err)
	require.Len(t, db.execCalls, 2)

	require.Contains(t, db.execCalls[0].query, "RETRYABLE")
	require.Equal(t, int32(5), db.execCalls[0].args[0])
	require.Equal(t, "first error", db.execCalls[0].args[1])
	require.Equal(t, pgtype.Interval{Microseconds: 2 * int64(time.Second/time.Microsecond), Valid: true}, db.execCalls[0].args[2])
	require.Equal(t, "1-paid", db.execCalls[0].args[3])

	require.Equal(t, "", db.execCalls[1].args[1])
	require.Equal(t, "2-cancelled", db.execCalls[1].args[3])
}

func TestInboxRepository_MarkAsFailedEmptyKeys(t *testing.T) {
	t.Parallel()

	db := &fakeDB{}
	repo := NewInboxRepository(db)

	err := repo.MarkAsFailed(context.Background(), nil, nil, 5, time.Second)

	require.NoError(t, err)
	require.Empty(t, db.execCalls)
}

func TestInboxRepository_MarkAsFailedLengthMismatch(t *testing.T) {
	t.Parallel()

	db := &fakeDB{}
	repo := NewInboxRepository(db)

	err := repo.MarkAsFailed(context.Background(), []string{"1-paid"}, nil, 5, time.Second)

	require.ErrorContains(t, err, "keys/errors length mismatch")
	require.Empty(t, db.execCalls)
}

func TestInboxRepository_MarkAsFailedError(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	repo := NewInboxRepository(&fakeDB{execErr: dbErr})

	err := repo.MarkAsFailed(context.Background(), []string{"1-paid"}, []error{errors.New("send failed")}, 5, time.Second)

	require.ErrorIs(t, err, dbErr)
}

type execCall struct {
	query string
	args  []any
}

type queryCall struct {
	query string
	args  []any
}

type fakeDB struct {
	execErr  error
	queryErr error
	rows     pgx.Rows

	execCalls  []execCall
	queryCalls []queryCall
}

func (f *fakeDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	f.execCalls = append(f.execCalls, execCall{
		query: query,
		args:  append([]any(nil), args...),
	})

	return pgconn.CommandTag{}, f.execErr
}

func (f *fakeDB) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	f.queryCalls = append(f.queryCalls, queryCall{
		query: query,
		args:  append([]any(nil), args...),
	})

	if f.queryErr != nil {
		return nil, f.queryErr
	}
	if f.rows != nil {
		return f.rows, nil
	}

	return &fakeRows{}, nil
}

func (f *fakeDB) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{}
}

type fakeRow struct{}

func (fakeRow) Scan(...any) error {
	return nil
}

type fakeRows struct {
	items  []Data
	index  int
	err    error
	closed bool
}

func (f *fakeRows) Close() {
	f.closed = true
}

func (f *fakeRows) Err() error {
	return f.err
}

func (f *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (f *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (f *fakeRows) Next() bool {
	if f.index >= len(f.items) {
		return false
	}

	f.index++
	return true
}

func (f *fakeRows) Scan(dest ...any) error {
	item := f.items[f.index-1]

	*dest[0].(*string) = item.IdempotencyKey
	*dest[1].(*[]byte) = item.Data

	return nil
}

func (f *fakeRows) Values() ([]any, error) {
	item := f.items[f.index-1]
	return []any{item.IdempotencyKey, item.Data}, nil
}

func (f *fakeRows) RawValues() [][]byte {
	return nil
}

func (f *fakeRows) Conn() *pgx.Conn {
	return nil
}
