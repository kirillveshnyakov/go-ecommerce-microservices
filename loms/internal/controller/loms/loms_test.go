package loms

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/loms/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	lomsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var serviceError = errors.New("service error")

func TestLomsServer_CreateOrder(t *testing.T) {
	t.Parallel()

	items := []entity.Item{{SKU: 10, Count: 2}}

	tests := []struct {
		name        string
		setup       func(*mocks.MocklomsService)
		wantOrderID int64
		wantCode    codes.Code
	}{
		{
			name: "success",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CreateOrder(gomock.Any(), entity.UserID(15), items).
					Return(entity.OrderID(43), nil)
			},
			wantOrderID: 43,
			wantCode:    codes.OK,
		},
		{
			name: "insufficient stock",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CreateOrder(gomock.Any(), entity.UserID(15), items).
					Return(entity.OrderID(0), lomsErrors.ErrInsufficientStock)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "product not found",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CreateOrder(gomock.Any(), entity.UserID(15), items).
					Return(entity.OrderID(0), lomsErrors.ErrProductNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "service error",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CreateOrder(gomock.Any(), entity.UserID(15), items).
					Return(entity.OrderID(0), serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lomsService := mocks.NewMocklomsService(ctrl)
			tt.setup(lomsService)

			server := NewLomsServer(lomsService, zap.NewNop())

			resp, err := server.CreateOrder(context.Background(), &lomsv1.CreateOrderRequest{
				UserId: 15,
				Items:  []*lomsv1.Item{{Sku: 10, Count: 2}},
			})
			require.Equal(t, tt.wantCode, status.Code(err))
			if tt.wantCode == codes.OK {
				require.Equal(t, tt.wantOrderID, resp.GetOrderId())
			}
		})
	}
}

func TestLomsServer_GetOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*mocks.MocklomsService)
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					GetOrder(gomock.Any(), entity.OrderID(43)).
					Return(&entity.Order{
						ID:     43,
						UserID: 15,
						Status: entity.OrderStatusAwaitingPayment,
						Items:  []entity.Item{{SKU: 10, Count: 2}},
					}, nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "not found",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					GetOrder(gomock.Any(), entity.OrderID(43)).
					Return(nil, lomsErrors.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "service error",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					GetOrder(gomock.Any(), entity.OrderID(43)).
					Return(nil, serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lomsService := mocks.NewMocklomsService(ctrl)
			tt.setup(lomsService)

			server := NewLomsServer(lomsService, zap.NewNop())

			resp, err := server.GetOrder(context.Background(), &lomsv1.GetOrderRequest{OrderId: 43})
			require.Equal(t, tt.wantCode, status.Code(err))
			if tt.wantCode == codes.OK {
				require.Equal(t, lomsv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT, resp.GetStatus())
			}
		})
	}
}

func TestLomsServer_PayOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*mocks.MocklomsService)
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					PayOrder(gomock.Any(), entity.OrderID(43)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "order not found",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					PayOrder(gomock.Any(), entity.OrderID(43)).
					Return(lomsErrors.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "invalid status",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					PayOrder(gomock.Any(), entity.OrderID(43)).
					Return(lomsErrors.ErrInvalidStatus)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "service error",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					PayOrder(gomock.Any(), entity.OrderID(43)).
					Return(serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lomsService := mocks.NewMocklomsService(ctrl)
			tt.setup(lomsService)

			server := NewLomsServer(lomsService, zap.NewNop())

			_, err := server.PayOrder(context.Background(), &lomsv1.PayOrderRequest{OrderId: 43})
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestLomsServer_CancelOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*mocks.MocklomsService)
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CancelOrder(gomock.Any(), entity.OrderID(43)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "order not found",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CancelOrder(gomock.Any(), entity.OrderID(43)).
					Return(lomsErrors.ErrOrderNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "product not found",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CancelOrder(gomock.Any(), entity.OrderID(43)).
					Return(lomsErrors.ErrProductNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "insufficient stock",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CancelOrder(gomock.Any(), entity.OrderID(43)).
					Return(lomsErrors.ErrInsufficientStock)
			},
			wantCode: codes.ResourceExhausted,
		},
		{
			name: "invalid status",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CancelOrder(gomock.Any(), entity.OrderID(43)).
					Return(lomsErrors.ErrInvalidStatus)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "service error",
			setup: func(lomsService *mocks.MocklomsService) {
				lomsService.EXPECT().
					CancelOrder(gomock.Any(), entity.OrderID(43)).
					Return(serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			lomsService := mocks.NewMocklomsService(ctrl)
			tt.setup(lomsService)

			server := NewLomsServer(lomsService, zap.NewNop())

			_, err := server.CancelOrder(context.Background(), &lomsv1.CancelOrderRequest{OrderId: 43})
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}
