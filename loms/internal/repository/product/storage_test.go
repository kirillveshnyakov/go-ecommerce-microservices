package product

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
	queryRow func(context.Context, string, ...interface{}) pgx.Row
}

func (db fakeDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (db fakeDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (db fakeDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.queryRow(ctx, sql, args...)
}

type fakeRow struct {
	values []interface{}
	err    error
}

func (r fakeRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	for i, value := range r.values {
		switch d := dest[i].(type) {
		case *int64:
			*d = value.(int64)
		case *string:
			*d = value.(string)
		}
	}
	return nil
}

func TestProductRepository_AddProduct(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name    string
		row     pgx.Row
		wantSKU entity.SKU
		wantErr error
	}{
		{
			name:    "success",
			row:     fakeRow{values: []interface{}{int64(16)}},
			wantSKU: 16,
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

			repository := NewProductRepository(fakeDB{
				queryRow: func(_ context.Context, _ string, args ...interface{}) pgx.Row {
					require.Equal(t, []interface{}{"phone", int64(1000)}, args)
					return tt.row
				},
			})

			gotSKU, err := repository.AddProduct(context.Background(), entity.Product{
				Name:  "phone",
				Price: 1000,
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSKU, gotSKU)
		})
	}
}

func TestProductRepository_GetProduct(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name        string
		row         pgx.Row
		wantProduct entity.Product
		wantErr     error
	}{
		{
			name: "success",
			row:  fakeRow{values: []interface{}{"phone", int64(1000)}},
			wantProduct: entity.Product{
				ID:    16,
				Name:  "phone",
				Price: 1000,
			},
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

			repository := NewProductRepository(fakeDB{
				queryRow: func(_ context.Context, _ string, args ...interface{}) pgx.Row {
					require.Equal(t, []interface{}{int64(16)}, args)
					return tt.row
				},
			})

			gotProduct, err := repository.GetProduct(context.Background(), 16)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantProduct, gotProduct)
		})
	}
}
