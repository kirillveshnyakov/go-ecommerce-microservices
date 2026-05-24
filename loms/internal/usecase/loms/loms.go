package loms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/outbox"
	"go.uber.org/zap"
)

//go:generate mockgen -source=loms.go -destination=mocks/loms_mocks.go -package=mocks
type (
	orderRepository interface {
		AddOrder(ctx context.Context, order *entity.Order) (entity.OrderID, error)
		GetOrder(ctx context.Context, orderID entity.OrderID) (*entity.Order, error)
		GetOrderWithLock(ctx context.Context, orderID entity.OrderID) (*entity.Order, error)
		SwapOrderStatus(_ context.Context, orderID entity.OrderID, oldStatus entity.OrderStatus, newStatus entity.OrderStatus) error
	}

	stocksRepository interface {
		ReserveStocks(ctx context.Context, orderID entity.OrderID, items []entity.Item) error
		ReleaseStocks(ctx context.Context, orderID entity.OrderID, items []entity.Item) error
	}

	outboxRepository interface {
		AddMessage(ctx context.Context, idempotencyKey string, kind outbox.Kind, message []byte) error
	}

	notificationsClient interface {
		SendOrderStatusChangedNotification(ctx context.Context, message port.Notification) error
	}

	transactor interface {
		WithTx(ctx context.Context, f func(ctx context.Context) error) (err error)
	}
)

type lomsService struct {
	orderRepository     orderRepository
	stocksRepository    stocksRepository
	outboxRepository    outboxRepository
	notificationsClient notificationsClient
	logger              *zap.Logger
	transactor          transactor
}

func NewLomsService(
	orderRepository orderRepository,
	stocksRepository stocksRepository,
	outboxRepository outboxRepository,
	notificationsClient notificationsClient,
	logger *zap.Logger,
	transactor transactor,
) *lomsService {
	return &lomsService{
		orderRepository:     orderRepository,
		stocksRepository:    stocksRepository,
		outboxRepository:    outboxRepository,
		notificationsClient: notificationsClient,
		logger:              logger,
		transactor:          transactor,
	}
}

func (s *lomsService) CreateOrder(ctx context.Context, userID entity.UserID, items []entity.Item) (entity.OrderID, error) {
	order := &entity.Order{
		UserID: userID,
		Items:  items,
		Status: entity.OrderStatusAwaitingPayment,
	}

	err := s.transactor.WithTx(ctx, func(ctx context.Context) error {
		id, err := s.orderRepository.AddOrder(ctx, order)
		if err != nil {
			return s.wrapCreateOrderError(err, userID, items)
		}
		order.ID = id
		if err = s.stocksRepository.ReserveStocks(ctx, order.ID, items); err != nil {
			if errors.Is(err, lomsErrors.ErrInsufficientStock) {
				return err
			}
			return s.wrapCreateOrderError(err, userID, items)
		}
		return s.createOutboxMessage(ctx, order)
	})
	if err != nil {
		return 0, s.wrapCreateOrderError(err, userID, items)
	}
	return order.ID, nil
}

func (s *lomsService) GetOrder(ctx context.Context, orderID entity.OrderID) (*entity.Order, error) {
	order, err := s.orderRepository.GetOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, lomsErrors.ErrOrderNotFound) {
			return nil, err
		}
		return nil, s.wrapGetOrderError(err, orderID)
	}
	return order, nil
}

func (s *lomsService) PayOrder(ctx context.Context, orderID entity.OrderID) error {
	return s.transactor.WithTx(ctx, func(ctx context.Context) error {
		order, err := s.orderRepository.GetOrderWithLock(ctx, orderID)
		if err != nil {
			if errors.Is(err, lomsErrors.ErrOrderNotFound) {
				return err
			}
			return s.wrapPayOrderError(err, orderID)
		}

		err = s.orderRepository.SwapOrderStatus(ctx, orderID, entity.OrderStatusAwaitingPayment, entity.OrderStatusPaid)
		if err != nil {
			if errors.Is(err, lomsErrors.ErrOrderNotFound) || errors.Is(err, lomsErrors.ErrInvalidStatus) {
				return err
			}
			return s.wrapPayOrderError(err, orderID)
		}
		order.Status = entity.OrderStatusPaid

		return s.createOutboxMessage(ctx, order)
	})
}

func (s *lomsService) CancelOrder(ctx context.Context, orderID entity.OrderID) error {
	return s.transactor.WithTx(ctx, func(ctx context.Context) error {
		order, err := s.orderRepository.GetOrderWithLock(ctx, orderID)
		if err != nil {
			if errors.Is(err, lomsErrors.ErrOrderNotFound) {
				return err
			}
			return s.wrapCancelOrderError(err, orderID)
		}

		err = s.orderRepository.SwapOrderStatus(ctx, orderID, entity.OrderStatusAwaitingPayment, entity.OrderStatusCancelled)
		if err != nil {
			if errors.Is(err, lomsErrors.ErrOrderNotFound) || errors.Is(err, lomsErrors.ErrInvalidStatus) {
				return err
			}
			return s.wrapCancelOrderError(err, orderID)
		}

		err = s.stocksRepository.ReleaseStocks(ctx, orderID, order.Items)
		if err != nil {
			if errors.Is(err, lomsErrors.ErrInsufficientStock) {
				return err
			}
			return s.wrapCancelOrderError(err, orderID)
		}
		order.Status = entity.OrderStatusCancelled

		return s.createOutboxMessage(ctx, order)
	})
}

func (s *lomsService) createIdempotencyKey(orderID entity.OrderID, status entity.OrderStatus) string {
	return strconv.FormatInt(int64(orderID), 10) + "-" + string(status)
}

func (s *lomsService) createOutboxMessage(ctx context.Context, order *entity.Order) error {
	key := s.createIdempotencyKey(order.ID, order.Status)

	message := port.Notification{
		UserID:  int64(order.UserID),
		OrderID: int64(order.ID),
		Status:  port.FromEntityToPortStatus(order.Status),
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("loms usecase - create outbox message: order=%v: %w", order, err)
	}

	return s.outboxRepository.AddMessage(ctx, key, outbox.KindNotification, body)
}

func (s *lomsService) OrderStatusChangedNotificationKindHandler(ctx context.Context, data []byte) error {
	var message port.Notification
	err := json.Unmarshal(data, &message)
	if err != nil {
		return fmt.Errorf("order status changed notification kind handler: %w", err)
	}

	return s.notificationsClient.SendOrderStatusChangedNotification(ctx, message)
}

func (s *lomsService) wrapCreateOrderError(err error, userID entity.UserID, items []entity.Item) error {
	s.logger.Error("loms usecase - create order failed",
		zap.Int64("user_id", int64(userID)),
		zap.String("items", fmt.Sprintf("%v", items)),
		zap.Error(err),
	)

	return fmt.Errorf("loms usecase - create order: user_id=%d items=%v: %w", userID, items, err)
}

func (s *lomsService) wrapGetOrderError(err error, orderID entity.OrderID) error {
	s.logger.Error("loms usecase - get order failed",
		zap.Int64("order_id", int64(orderID)),
		zap.Error(err),
	)

	return fmt.Errorf("loms usecase - get order: order_id=%d: %w", orderID, err)
}

func (s *lomsService) wrapPayOrderError(err error, orderID entity.OrderID) error {
	s.logger.Error("loms usecase - pay order failed",
		zap.Int64("order_id", int64(orderID)),
		zap.Error(err),
	)

	return fmt.Errorf("loms usecase - pay order: order_id=%d: %w", orderID, err)
}

func (s *lomsService) wrapCancelOrderError(err error, orderID entity.OrderID) error {
	s.logger.Error("loms usecase - cancel order failed",
		zap.Int64("order_id", int64(orderID)),
		zap.Error(err),
	)

	return fmt.Errorf("loms usecase - cancel order: order_id=%d: %w", orderID, err)
}
