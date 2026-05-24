package cart

import (
	"context"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/repository/cart/sqlc"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5"
)

type (
	DB interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		sqlc.DBTX
	}
)

type cartRepository struct {
	queries    *sqlc.Queries
	transactor transactor.Transactor
}

func NewCartRepository(db DB) *cartRepository {
	return &cartRepository{
		queries:    sqlc.New(db),
		transactor: transactor.NewTransactor(db),
	}
}

func (r *cartRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *cartRepository) AddItemChecked(ctx context.Context, userID entity.UserID, sku entity.SKU, count uint32, stock uint64) error {
	rows, err := r.getQueries(ctx).AddItemInCartChecked(ctx, sqlc.AddItemInCartCheckedParams{
		UserID: int64(userID),
		Sku:    int64(sku),
		Amount: int64(count),
		Stock:  int64(stock),
	})
	if err != nil {
		return fmt.Errorf("cart repository - add item: user_id=%d sku=%d count=%d: %w", userID, sku, count, err)
	}
	if rows == 0 {
		return fmt.Errorf("cart repository - add item: user_id=%d sku=%d count=%d: %w", userID, sku, count, cartErrors.ErrInsufficientStock)
	}
	return nil
}

func (r *cartRepository) DeleteItem(ctx context.Context, userID entity.UserID, sku entity.SKU) error {
	err := r.getQueries(ctx).DeleteItemFromCart(ctx, sqlc.DeleteItemFromCartParams{
		UserID: int64(userID),
		Sku:    int64(sku),
	})
	if err != nil {
		return fmt.Errorf("cart repository - delete item: user_id=%d sku=%d: %w", userID, sku, err)
	}
	return nil
}

func (r *cartRepository) ClearUserCart(ctx context.Context, userID entity.UserID) error {
	err := r.getQueries(ctx).ClearUserCart(ctx, int64(userID))
	if err != nil {
		return fmt.Errorf("cart repository - clear cart: user_id=%d: %w", userID, err)
	}
	return nil
}

func (r *cartRepository) GetCart(ctx context.Context, userID entity.UserID) ([]entity.Item, error) {
	dbItems, err := r.getQueries(ctx).GetUserCart(ctx, int64(userID))
	if err != nil {
		return []entity.Item{}, fmt.Errorf("cart repository - get cart: user_id=%d: %w", userID, err)
	}

	if len(dbItems) == 0 {
		return []entity.Item{}, fmt.Errorf("cart repository - get cart: user_id=%d: %w", userID, cartErrors.ErrUserCartNotFound)
	}

	items := make([]entity.Item, len(dbItems))
	for i, dbItem := range dbItems {
		items[i] = entity.Item{
			SKU:   entity.SKU(dbItem.Sku),
			Count: uint32(dbItem.Amount),
		}
	}

	return items, nil
}

func (r *cartRepository) GetCartWithLock(ctx context.Context, userID entity.UserID) ([]entity.Item, error) {
	dbItems, err := r.getQueries(ctx).GetUserCartForUpdate(ctx, int64(userID))
	if err != nil {
		return []entity.Item{}, fmt.Errorf("cart repository - get cart: user_id=%d: %w", userID, err)
	}

	if len(dbItems) == 0 {
		return []entity.Item{}, fmt.Errorf("cart repository - get cart: user_id=%d: %w", userID, cartErrors.ErrUserCartNotFound)
	}

	items := make([]entity.Item, len(dbItems))
	for i, dbItem := range dbItems {
		items[i] = entity.Item{
			SKU:   entity.SKU(dbItem.Sku),
			Count: uint32(dbItem.Amount),
		}
	}

	return items, nil
}

func (r *cartRepository) GetItemCount(ctx context.Context, userID entity.UserID, sku entity.SKU) (uint32, error) {
	count, err := r.getQueries(ctx).GetItemCountInCart(ctx, sqlc.GetItemCountInCartParams{
		UserID: int64(userID),
		Sku:    int64(sku),
	})
	if err != nil {
		return 0, fmt.Errorf("cart repository - get cart: user_id=%d: %w", userID, err)
	}
	return uint32(count), nil
}
