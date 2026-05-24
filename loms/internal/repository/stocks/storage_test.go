package stocks

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

type fakeDB struct {
	exec     func(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	queryRow func(context.Context, string, ...interface{}) pgx.Row
}

func (db fakeDB) Begin(context.Context) (pgx.Tx, error) {
	return nil, nil
}

func (db fakeDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.exec(ctx, sql, args...)
}

func (db fakeDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (db fakeDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.queryRow(ctx, sql, args...)
}

type fakeRow struct {
	value int64
	err   error
}

func (r fakeRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*int64)) = r.value
	return nil
}

func TestStocksRepository_GetStock(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name      string
		row       pgx.Row
		wantCount uint64
		wantErr   error
	}{
		{
			name:      "success",
			row:       fakeRow{value: 5},
			wantCount: 5,
		},
		{
			name:    "not found",
			row:     fakeRow{err: pgx.ErrNoRows},
			wantErr: lomsErrors.ErrProductNotFound,
		},
		{
			name:    "query error",
			row:     fakeRow{err: repositoryErr},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := NewStocksRepository(fakeDB{
				exec: func(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
					return pgconn.CommandTag{}, nil
				},
				queryRow: func(_ context.Context, _ string, args ...interface{}) pgx.Row {
					require.Equal(t, []interface{}{int64(15)}, args)
					return tt.row
				},
			})

			gotCount, err := repository.GetStock(context.Background(), 15)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantCount, gotCount)
		})
	}
}

func TestStocksRepository_SetStock(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name    string
		tag     pgconn.CommandTag
		err     error
		wantErr error
	}{
		{
			name: "success",
			tag:  pgconn.NewCommandTag("INSERT 0 1"),
		},
		{
			name:    "not found",
			tag:     pgconn.NewCommandTag("INSERT 0 0"),
			wantErr: lomsErrors.ErrProductNotFound,
		},
		{
			name:    "exec error",
			err:     repositoryErr,
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := NewStocksRepository(fakeDB{
				exec: func(_ context.Context, _ string, args ...interface{}) (pgconn.CommandTag, error) {
					require.Equal(t, []interface{}{int64(15), int64(5)}, args)
					return tt.tag, tt.err
				},
				queryRow: func(context.Context, string, ...interface{}) pgx.Row {
					return fakeRow{}
				},
			})

			err := repository.SetStock(context.Background(), entity.SKU(15), 5)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
