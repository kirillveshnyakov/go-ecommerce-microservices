package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/jackc/pgx/v5/pgconn"

	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/order/sqlc"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5"
)

type (
	DB interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		sqlc.DBTX
	}
)

const (
	PGErrForeignKey = "23503"
)

type ordersRepository struct {
	queries    *sqlc.Queries
	transactor transactor.Transactor
}

func NewOrdersRepository(db DB) *ordersRepository {
	return &ordersRepository{
		queries:    sqlc.New(db),
		transactor: transactor.NewTransactor(db),
	}
}

func (r *ordersRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func mergingDuplicates(items []entity.Item) []entity.Item {
	merged := make(map[entity.SKU]uint64, len(items))
	for _, item := range items {
		merged[item.SKU] += item.Count
	}

	result := make([]entity.Item, 0, len(merged))
	for sku, count := range merged {
		result = append(result, entity.Item{
			SKU:   sku,
			Count: count,
		})
	}

	return result
}

func (r *ordersRepository) AddOrder(ctx context.Context, order *entity.Order) (entity.OrderID, error) {
	var orderID entity.OrderID
	err := r.transactor.WithTx(ctx, func(ctx context.Context) error {
		queries := r.getQueries(ctx)

		dbOrderID, err := queries.AddOrder(ctx, sqlc.AddOrderParams{
			UserID: int64(order.UserID),
			Status: FromEntityStatus(order.Status),
		})
		if err != nil {
			return fmt.Errorf("order repository - add order: order=%v: %w", order, err)
		}

		items := mergingDuplicates(order.Items)
		for _, item := range items {
			err = queries.AddOrderInfo(ctx, sqlc.AddOrderInfoParams{
				OrderID: dbOrderID,
				Sku:     int64(item.SKU),
				Amount:  int64(item.Count),
			})
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) {
					if pgErr.Code == PGErrForeignKey {
						switch pgErr.ConstraintName {
						case "order_info_order_id_fkey":
							return fmt.Errorf("order repository - add order: order=%v: %w", order, lomsErrors.ErrOrderNotFound)
						case "order_info_sku_fkey":
							return fmt.Errorf("order repository - add order: order=%v: %w", order, lomsErrors.ErrProductNotFound)
						}
					}
				}
				return fmt.Errorf("order repository - add order: order=%v: %w", order, err)
			}
		}
		orderID = entity.OrderID(dbOrderID)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return orderID, nil
}

func (r *ordersRepository) GetOrder(ctx context.Context, orderID entity.OrderID) (*entity.Order, error) {
	queries := r.getQueries(ctx)
	orderInfo, err := queries.GetOrder(ctx, int64(orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order repository - get order: order_id=%d: %w", orderID, lomsErrors.ErrOrderNotFound)
		}
		return nil, fmt.Errorf("order repository - get order: order_id=%d: %w", orderID, err)
	}

	orderItems, err := queries.GetOrderItems(ctx, int64(orderID))
	if err != nil {
		return nil, fmt.Errorf("order repository - get order: order_id=%d: %w", orderID, err)
	}

	items := make([]entity.Item, len(orderItems))
	for i, item := range orderItems {
		items[i] = entity.Item{
			SKU:   entity.SKU(item.Sku),
			Count: uint64(item.Amount),
		}
	}

	return &entity.Order{
		ID:        orderID,
		UserID:    entity.UserID(orderInfo.UserID),
		Status:    ToEntityStatus(orderInfo.Status),
		Items:     items,
		CreatedAt: orderInfo.CreatedAt.Time,
		UpdatedAt: orderInfo.UpdatedAt.Time,
	}, nil
}

func (r *ordersRepository) GetOrderWithLock(ctx context.Context, orderID entity.OrderID) (*entity.Order, error) {
	queries := r.getQueries(ctx)
	orderInfo, err := queries.GetOrderForUpdate(ctx, int64(orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("order repository - get order: order_id=%d: %w", orderID, lomsErrors.ErrOrderNotFound)
		}
		return nil, fmt.Errorf("order repository - get order: order_id=%d: %w", orderID, err)
	}

	orderItems, err := queries.GetOrderItems(ctx, int64(orderID))
	if err != nil {
		return nil, fmt.Errorf("order repository - get order: order_id=%d: %w", orderID, err)
	}

	items := make([]entity.Item, len(orderItems))
	for i, item := range orderItems {
		items[i] = entity.Item{
			SKU:   entity.SKU(item.Sku),
			Count: uint64(item.Amount),
		}
	}

	return &entity.Order{
		ID:        orderID,
		UserID:    entity.UserID(orderInfo.UserID),
		Status:    ToEntityStatus(orderInfo.Status),
		Items:     items,
		CreatedAt: orderInfo.CreatedAt.Time,
		UpdatedAt: orderInfo.UpdatedAt.Time,
	}, nil
}

func (r *ordersRepository) GetOrderStatus(ctx context.Context, orderID entity.OrderID) (entity.OrderStatus, error) {
	status, err := r.getQueries(ctx).GetOrderStatus(ctx, int64(orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("order repository - get order status: order_id=%d: %w", orderID, lomsErrors.ErrOrderNotFound)
		}
		return "", fmt.Errorf("order repository - get order status: order_id=%d: %w", orderID, err)
	}
	return ToEntityStatus(status), nil
}

func (r *ordersRepository) SwapOrderStatus(ctx context.Context, orderID entity.OrderID, oldStatus entity.OrderStatus, newStatus entity.OrderStatus) error {
	queries := r.getQueries(ctx)
	rows, err := queries.SwapOrderStatus(ctx, sqlc.SwapOrderStatusParams{
		ID:       int64(orderID),
		Status:   FromEntityStatus(oldStatus),
		Status_2: FromEntityStatus(newStatus),
	})
	if err != nil {
		return fmt.Errorf("order repository - swap order status: order_id=%d old_status=%s new_status=%s : %w",
			orderID, oldStatus, newStatus, err)
	}
	if rows == 0 {
		_, err = queries.GetOrderStatus(ctx, int64(orderID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("order repository - swap order status: order_id=%d old_status=%s new_status=%s : %w",
					orderID, oldStatus, newStatus, lomsErrors.ErrOrderNotFound)
			}
			return fmt.Errorf("order repository - swap order status: order_id=%d old_status=%s new_status=%s : %w",
				orderID, oldStatus, newStatus, err)
		}
		return fmt.Errorf("order repository - swap order status: order_id=%d old_status=%s new_status=%s : %w",
			orderID, oldStatus, newStatus, lomsErrors.ErrInvalidStatus)
	}
	return nil
}

func ToEntityStatus(status sqlc.LomsOrderStatus) entity.OrderStatus {
	switch status {
	case sqlc.LomsOrderStatusNew:
		return entity.OrderStatusNew
	case sqlc.LomsOrderStatusAwaitingPayment:
		return entity.OrderStatusAwaitingPayment
	case sqlc.LomsOrderStatusFailed:
		return entity.OrderStatusFailed
	case sqlc.LomsOrderStatusPaid:
		return entity.OrderStatusPaid
	case sqlc.LomsOrderStatusCancelled:
		return entity.OrderStatusCancelled
	}
	return entity.OrderStatusUnavailable
}

func FromEntityStatus(status entity.OrderStatus) sqlc.LomsOrderStatus {
	switch status {
	case entity.OrderStatusNew:
		return sqlc.LomsOrderStatusNew
	case entity.OrderStatusAwaitingPayment:
		return sqlc.LomsOrderStatusAwaitingPayment
	case entity.OrderStatusFailed:
		return sqlc.LomsOrderStatusFailed
	case entity.OrderStatusPaid:
		return sqlc.LomsOrderStatusPaid
	case entity.OrderStatusCancelled:
		return sqlc.LomsOrderStatusCancelled
	}
	return sqlc.LomsOrderStatusUnavailable
}
