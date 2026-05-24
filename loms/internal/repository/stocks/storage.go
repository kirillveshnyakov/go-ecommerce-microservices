package stocks

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/stocks/sqlc"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5"
)

type (
	DB interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		sqlc.DBTX
	}
)

type stocksRepository struct {
	queries    *sqlc.Queries
	transactor transactor.Transactor
}

func NewStocksRepository(db DB) *stocksRepository {
	return &stocksRepository{
		queries:    sqlc.New(db),
		transactor: transactor.NewTransactor(db),
	}
}

func (r *stocksRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *stocksRepository) GetStock(ctx context.Context, sku entity.SKU) (uint64, error) {
	stock, err := r.getQueries(ctx).GetAvailableStock(ctx, int64(sku))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("product repository - get stock: sku=%d: %w", sku, lomsErrors.ErrProductNotFound)
		}
		return 0, fmt.Errorf("product repository - get stock: sku=%d: %w", sku, err)
	}

	return uint64(stock), nil
}

func (r *stocksRepository) SetStock(ctx context.Context, sku entity.SKU, count uint64) error {
	rows, err := r.getQueries(ctx).SetAvailableStock(ctx, sqlc.SetAvailableStockParams{
		Sku:    int64(sku),
		Amount: int64(count),
	})
	if err != nil {
		return fmt.Errorf("product repository - set stock: sku=%d count=%d: %w", sku, count, err)
	}
	if rows == 0 {
		return fmt.Errorf("product repository - set stock: sku=%d count=%d: %w", sku, count, lomsErrors.ErrProductNotFound)
	}
	return nil
}

func (r *stocksRepository) ReserveStocks(ctx context.Context, orderID entity.OrderID, items []entity.Item) error {
	return r.transactor.WithTx(ctx, func(ctx context.Context) error {
		queries := r.getQueries(ctx)

		for _, item := range items {
			rows, err := queries.DecrementAvailableStock(ctx, sqlc.DecrementAvailableStockParams{
				Sku:    int64(item.SKU),
				Amount: int64(item.Count),
			})
			if err != nil {
				return fmt.Errorf("stocks repository - reserve stocks: decrement available stock sku=%d count=%d: %w",
					item.SKU, item.Count, err)
			}
			if rows == 0 {
				_, err = queries.GetAvailableStock(ctx, int64(item.SKU))
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return fmt.Errorf(
							"stocks repository - reserve stocks: decrement available stock sku=%d count=%d: %w",
							item.SKU, item.Count, lomsErrors.ErrProductNotFound,
						)
					}
					return fmt.Errorf(
						"stocks repository - reserve stocks: check product exist sku=%d count=%d: %w",
						item.SKU, item.Count, err,
					)
				}

				return fmt.Errorf("stocks repository - reserve stocks: decrement available stock sku=%d count=%d: %w",
					item.SKU, item.Count, lomsErrors.ErrInsufficientStock)
			}

			if err = queries.AddReserveStock(ctx, sqlc.AddReserveStockParams{
				Sku:     int64(item.SKU),
				OrderID: int64(orderID),
				Amount:  int64(item.Count),
			}); err != nil {
				return fmt.Errorf("stocks repository - reserve stocks: add reserved stock sku=%d order_id=%d count=%d: %w",
					item.SKU, orderID, item.Count, err)
			}
		}

		return nil
	})
}

func (r *stocksRepository) ReleaseStocks(ctx context.Context, orderID entity.OrderID, items []entity.Item) error {
	return r.transactor.WithTx(ctx, func(ctx context.Context) error {
		queries := r.getQueries(ctx)

		for _, item := range items {
			rows, err := queries.DecrementReservedStock(ctx, sqlc.DecrementReservedStockParams{
				Sku:     int64(item.SKU),
				OrderID: int64(orderID),
				Amount:  int64(item.Count),
			})
			if err != nil {
				return fmt.Errorf("stocks repository - release stocks: decrement reserved stock sku=%d order_id=%d count=%d: %w",
					item.SKU, orderID, item.Count, err)
			}
			if rows == 0 {
				_, err = queries.GetAvailableStock(ctx, int64(item.SKU))
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return fmt.Errorf(
							"stocks repository - release stocks: decrement reserved stock sku=%d count=%d: %w",
							item.SKU, item.Count, lomsErrors.ErrProductNotFound,
						)
					}
					return fmt.Errorf(
						"stocks repository - release stocks: check product exist sku=%d count=%d: %w",
						item.SKU, item.Count, err,
					)
				}

				return fmt.Errorf("stocks repository - release stocks: decrement reserved stock sku=%d order_id=%d count=%d: %w",
					item.SKU, orderID, item.Count, lomsErrors.ErrInsufficientStock)
			}

			if err = queries.AddAvailableStock(ctx, sqlc.AddAvailableStockParams{
				Sku:    int64(item.SKU),
				Amount: int64(item.Count),
			}); err != nil {
				return fmt.Errorf("stocks repository - release stocks: add available stock sku=%d count=%d: %w",
					item.SKU, item.Count, err)
			}
		}

		return nil
	})
}
