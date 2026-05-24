package loms

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/outbox"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usecase/loms/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestLomsService_CreateOrder(t *testing.T) {
	t.Parallel()

	items := []entity.Item{
		{SKU: 4, Count: 2},
		{SKU: 8, Count: 1},
	}
	repositoryErr := errors.New("repository error")
	outboxErr := errors.New("outbox error")

	tests := []struct {
		name        string
		setup       func(*testing.T, *mocks.MockorderRepository, *mocks.MockstocksRepository, *mocks.MockoutboxRepository, *mocks.Mocktransactor)
		wantOrderID entity.OrderID
		wantErr     error
	}{
		{
			name: "success",
			setup: func(t *testing.T, orderRepository *mocks.MockorderRepository, stocksRepository *mocks.MockstocksRepository, outboxRepository *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()).
					Return(entity.OrderID(777), nil)
				stocksRepository.EXPECT().
					ReserveStocks(gomock.Any(), entity.OrderID(777), items).
					Return(nil)
				outboxRepository.EXPECT().
					AddMessage(gomock.Any(), "777-awaiting_payment", outbox.KindNotification, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ outbox.Kind, message []byte) error {
						assertNotification(t, message, port.Notification{
							UserID:  42,
							OrderID: 777,
							Status:  port.OrderStatusAwaitingPayment,
						})
						return nil
					})
			},
			wantOrderID: 777,
		},
		{
			name: "insufficient stock",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, stocksRepository *mocks.MockstocksRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()).
					Return(entity.OrderID(777), nil)
				stocksRepository.EXPECT().
					ReserveStocks(gomock.Any(), entity.OrderID(777), items).
					Return(lomsErrors.ErrInsufficientStock)
			},
			wantErr: lomsErrors.ErrInsufficientStock,
		},
		{
			name: "add order error",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, _ *mocks.MockstocksRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()).
					Return(entity.OrderID(0), repositoryErr)
			},
			wantErr: repositoryErr,
		},
		{
			name: "reserve stocks error",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, stocksRepository *mocks.MockstocksRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()).
					Return(entity.OrderID(777), nil)
				stocksRepository.EXPECT().
					ReserveStocks(gomock.Any(), entity.OrderID(777), items).
					Return(repositoryErr)
			},
			wantErr: repositoryErr,
		},
		{
			name: "outbox error",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, stocksRepository *mocks.MockstocksRepository, outboxRepository *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					AddOrder(gomock.Any(), gomock.Any()).
					Return(entity.OrderID(777), nil)
				stocksRepository.EXPECT().
					ReserveStocks(gomock.Any(), entity.OrderID(777), items).
					Return(nil)
				outboxRepository.EXPECT().
					AddMessage(gomock.Any(), "777-awaiting_payment", outbox.KindNotification, gomock.Any()).
					Return(outboxErr)
			},
			wantErr: outboxErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			orderRepository := mocks.NewMockorderRepository(ctrl)
			stocksRepository := mocks.NewMockstocksRepository(ctrl)
			outboxRepository := mocks.NewMockoutboxRepository(ctrl)
			transactor := mocks.NewMocktransactor(ctrl)

			tt.setup(t, orderRepository, stocksRepository, outboxRepository, transactor)

			service := NewLomsService(orderRepository, stocksRepository, outboxRepository, nil, zap.NewNop(), transactor)

			orderID, err := service.CreateOrder(context.Background(), 42, items)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantOrderID, orderID)
		})
	}
}

func TestLomsService_GetOrder(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name      string
		setup     func(*mocks.MockorderRepository)
		wantOrder *entity.Order
		wantErr   error
	}{
		{
			name: "success",
			setup: func(orderRepository *mocks.MockorderRepository) {
				orderRepository.EXPECT().
					GetOrder(gomock.Any(), entity.OrderID(777)).
					Return(&entity.Order{
						ID:     777,
						UserID: 42,
						Status: entity.OrderStatusAwaitingPayment,
						Items:  []entity.Item{{SKU: 4, Count: 2}},
					}, nil)
			},
			wantOrder: &entity.Order{
				ID:     777,
				UserID: 42,
				Status: entity.OrderStatusAwaitingPayment,
				Items:  []entity.Item{{SKU: 4, Count: 2}},
			},
		},
		{
			name: "not found",
			setup: func(orderRepository *mocks.MockorderRepository) {
				orderRepository.EXPECT().
					GetOrder(gomock.Any(), entity.OrderID(777)).
					Return(nil, lomsErrors.ErrOrderNotFound)
			},
			wantErr: lomsErrors.ErrOrderNotFound,
		},
		{
			name: "repository error",
			setup: func(orderRepository *mocks.MockorderRepository) {
				orderRepository.EXPECT().
					GetOrder(gomock.Any(), entity.OrderID(777)).
					Return(nil, repositoryErr)
			},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			orderRepository := mocks.NewMockorderRepository(ctrl)

			tt.setup(orderRepository)

			service := NewLomsService(orderRepository, nil, nil, nil, zap.NewNop(), nil)

			order, err := service.GetOrder(context.Background(), 777)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantOrder, order)
		})
	}
}

func TestLomsService_PayOrder(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name    string
		setup   func(*testing.T, *mocks.MockorderRepository, *mocks.MockoutboxRepository, *mocks.Mocktransactor)
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, orderRepository *mocks.MockorderRepository, outboxRepository *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(&entity.Order{
						ID:     777,
						UserID: 42,
						Status: entity.OrderStatusAwaitingPayment,
					}, nil)
				orderRepository.EXPECT().
					SwapOrderStatus(
						gomock.Any(),
						entity.OrderID(777),
						entity.OrderStatusAwaitingPayment,
						entity.OrderStatusPaid,
					).
					Return(nil)
				outboxRepository.EXPECT().
					AddMessage(gomock.Any(), "777-paid", outbox.KindNotification, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ outbox.Kind, message []byte) error {
						assertNotification(t, message, port.Notification{
							UserID:  42,
							OrderID: 777,
							Status:  port.OrderStatusPaid,
						})
						return nil
					})
			},
		},
		{
			name: "invalid status",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(&entity.Order{
						ID:     777,
						UserID: 42,
						Status: entity.OrderStatusCancelled,
					}, nil)
				orderRepository.EXPECT().
					SwapOrderStatus(
						gomock.Any(),
						entity.OrderID(777),
						entity.OrderStatusAwaitingPayment,
						entity.OrderStatusPaid,
					).
					Return(lomsErrors.ErrInvalidStatus)
			},
			wantErr: lomsErrors.ErrInvalidStatus,
		},
		{
			name: "not found",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(nil, lomsErrors.ErrOrderNotFound)
			},
			wantErr: lomsErrors.ErrOrderNotFound,
		},
		{
			name: "repository error",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(nil, repositoryErr)
			},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			orderRepository := mocks.NewMockorderRepository(ctrl)
			outboxRepository := mocks.NewMockoutboxRepository(ctrl)
			transactor := mocks.NewMocktransactor(ctrl)

			tt.setup(t, orderRepository, outboxRepository, transactor)

			service := NewLomsService(orderRepository, nil, outboxRepository, nil, zap.NewNop(), transactor)

			err := service.PayOrder(context.Background(), 777)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLomsService_CancelOrder(t *testing.T) {
	t.Parallel()

	items := []entity.Item{{SKU: 4, Count: 2}}
	repositoryErr := errors.New("repository error")

	tests := []struct {
		name    string
		setup   func(*testing.T, *mocks.MockorderRepository, *mocks.MockstocksRepository, *mocks.MockoutboxRepository, *mocks.Mocktransactor)
		wantErr error
	}{
		{
			name: "success",
			setup: func(t *testing.T, orderRepository *mocks.MockorderRepository, stocksRepository *mocks.MockstocksRepository, outboxRepository *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(&entity.Order{
						ID:     777,
						UserID: 42,
						Status: entity.OrderStatusAwaitingPayment,
						Items:  items,
					}, nil)
				orderRepository.EXPECT().
					SwapOrderStatus(
						gomock.Any(),
						entity.OrderID(777),
						entity.OrderStatusAwaitingPayment,
						entity.OrderStatusCancelled,
					).
					Return(nil)
				stocksRepository.EXPECT().
					ReleaseStocks(gomock.Any(), entity.OrderID(777), items).
					Return(nil)
				outboxRepository.EXPECT().
					AddMessage(gomock.Any(), "777-cancelled", outbox.KindNotification, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ outbox.Kind, message []byte) error {
						assertNotification(t, message, port.Notification{
							UserID:  42,
							OrderID: 777,
							Status:  port.OrderStatusCancelled,
						})
						return nil
					})
			},
		},
		{
			name: "order not found",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, _ *mocks.MockstocksRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(&entity.Order{}, lomsErrors.ErrOrderNotFound)
			},
			wantErr: lomsErrors.ErrOrderNotFound,
		},
		{
			name: "invalid status",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, _ *mocks.MockstocksRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(&entity.Order{
						ID:     777,
						UserID: 42,
						Status: entity.OrderStatusPaid,
						Items:  items,
					}, nil)
				orderRepository.EXPECT().
					SwapOrderStatus(
						gomock.Any(),
						entity.OrderID(777),
						entity.OrderStatusAwaitingPayment,
						entity.OrderStatusCancelled,
					).
					Return(lomsErrors.ErrInvalidStatus)
			},
			wantErr: lomsErrors.ErrInvalidStatus,
		},
		{
			name: "repository error",
			setup: func(_ *testing.T, orderRepository *mocks.MockorderRepository, _ *mocks.MockstocksRepository, _ *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(nil, repositoryErr)
			},
			wantErr: repositoryErr,
		},
		{
			name: "stock repository error",
			setup: func(t *testing.T, orderRepository *mocks.MockorderRepository, stocksRepository *mocks.MockstocksRepository, outboxRepository *mocks.MockoutboxRepository, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				orderRepository.EXPECT().
					GetOrderWithLock(gomock.Any(), entity.OrderID(777)).
					Return(&entity.Order{
						ID:     777,
						UserID: 42,
						Status: entity.OrderStatusAwaitingPayment,
						Items:  items,
					}, nil)
				orderRepository.EXPECT().
					SwapOrderStatus(
						gomock.Any(),
						entity.OrderID(777),
						entity.OrderStatusAwaitingPayment,
						entity.OrderStatusCancelled,
					).
					Return(nil)
				stocksRepository.EXPECT().
					ReleaseStocks(gomock.Any(), entity.OrderID(777), items).
					Return(repositoryErr)
			},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			orderRepository := mocks.NewMockorderRepository(ctrl)
			stocksRepository := mocks.NewMockstocksRepository(ctrl)
			outboxRepository := mocks.NewMockoutboxRepository(ctrl)
			transactor := mocks.NewMocktransactor(ctrl)

			tt.setup(t, orderRepository, stocksRepository, outboxRepository, transactor)

			service := NewLomsService(orderRepository, stocksRepository, outboxRepository, nil, zap.NewNop(), transactor)

			err := service.CancelOrder(context.Background(), 777)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestLomsService_OrderStatusChangedNotificationKindHandler(t *testing.T) {
	t.Parallel()

	clientErr := errors.New("client error")

	tests := []struct {
		name          string
		data          []byte
		setup         func(*mocks.MocknotificationsClient)
		wantErr       error
		wantSyntaxErr bool
	}{
		{
			name: "success",
			data: []byte(`{"user_id":42,"order_id":777,"status":"paid"}`),
			setup: func(client *mocks.MocknotificationsClient) {
				client.EXPECT().
					SendOrderStatusChangedNotification(gomock.Any(), port.Notification{
						UserID:  42,
						OrderID: 777,
						Status:  port.OrderStatusPaid,
					}).
					Return(nil)
			},
		},
		{
			name:          "invalid json",
			data:          []byte(`{`),
			setup:         func(_ *mocks.MocknotificationsClient) {},
			wantSyntaxErr: true,
		},
		{
			name: "client error",
			data: []byte(`{"user_id":42,"order_id":777,"status":"cancelled"}`),
			setup: func(client *mocks.MocknotificationsClient) {
				client.EXPECT().
					SendOrderStatusChangedNotification(gomock.Any(), port.Notification{
						UserID:  42,
						OrderID: 777,
						Status:  port.OrderStatusCancelled,
					}).
					Return(clientErr)
			},
			wantErr: clientErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			notificationsClient := mocks.NewMocknotificationsClient(ctrl)

			tt.setup(notificationsClient)

			service := NewLomsService(nil, nil, nil, notificationsClient, zap.NewNop(), nil)

			err := service.OrderStatusChangedNotificationKindHandler(context.Background(), tt.data)
			if tt.wantErr == nil && !tt.wantSyntaxErr {
				require.NoError(t, err)
				return
			}
			if tt.wantSyntaxErr {
				var syntaxErr *json.SyntaxError
				require.ErrorAs(t, err, &syntaxErr)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func assertNotification(t *testing.T, data []byte, want port.Notification) {
	t.Helper()

	var got port.Notification
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got)
}
