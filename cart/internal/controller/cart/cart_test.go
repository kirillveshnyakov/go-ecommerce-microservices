package cart

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/controller/cart/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	cartv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var serviceError = errors.New("service error")

func TestCartServer_AddItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*mocks.MockitemService)
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(itemService *mocks.MockitemService) {
				itemService.EXPECT().
					AddItem(gomock.Any(), entity.UserID(3), entity.SKU(10), uint32(2)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "product not found",
			setup: func(itemService *mocks.MockitemService) {
				itemService.EXPECT().
					AddItem(gomock.Any(), entity.UserID(3), entity.SKU(10), uint32(2)).
					Return(cartErrors.ErrProductNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "insufficient stock",
			setup: func(itemService *mocks.MockitemService) {
				itemService.EXPECT().
					AddItem(gomock.Any(), entity.UserID(3), entity.SKU(10), uint32(2)).
					Return(cartErrors.ErrInsufficientStock)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "service error",
			setup: func(itemService *mocks.MockitemService) {
				itemService.EXPECT().
					AddItem(gomock.Any(), entity.UserID(3), entity.SKU(10), uint32(2)).
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
			itemService := mocks.NewMockitemService(ctrl)
			tt.setup(itemService)

			server := NewCartServer(itemService, nil, zap.NewNop())

			_, err := server.AddItem(context.Background(), &cartv1.AddItemRequest{
				UserId: 3,
				Sku:    10,
				Count:  2,
			})
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestCartServer_DeleteItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*mocks.MockitemService)
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(itemService *mocks.MockitemService) {
				itemService.EXPECT().
					DeleteItem(gomock.Any(), entity.UserID(3), entity.SKU(10)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "service error",
			setup: func(itemService *mocks.MockitemService) {
				itemService.EXPECT().
					DeleteItem(gomock.Any(), entity.UserID(3), entity.SKU(10)).
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
			itemService := mocks.NewMockitemService(ctrl)
			tt.setup(itemService)

			server := NewCartServer(itemService, nil, zap.NewNop())

			_, err := server.DeleteItem(context.Background(), &cartv1.DeleteItemRequest{
				UserId: 3,
				Sku:    10,
			})
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestCartServer_ClearCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*mocks.MockcartService)
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					ClearCart(gomock.Any(), entity.UserID(3)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "service error",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					ClearCart(gomock.Any(), entity.UserID(3)).
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
			cartService := mocks.NewMockcartService(ctrl)
			tt.setup(cartService)

			server := NewCartServer(nil, cartService, zap.NewNop())

			_, err := server.ClearCart(context.Background(), &cartv1.ClearCartRequest{UserId: 3})
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestCartServer_CheckoutCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(*mocks.MockcartService)
		wantOrderID int64
		wantCode    codes.Code
	}{
		{
			name: "success",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					CheckoutCart(gomock.Any(), entity.UserID(3)).
					Return(int64(99), nil)
			},
			wantOrderID: 99,
			wantCode:    codes.OK,
		},
		{
			name: "empty cart",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					CheckoutCart(gomock.Any(), entity.UserID(3)).
					Return(int64(0), cartErrors.ErrUserCartNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "product not found",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					CheckoutCart(gomock.Any(), entity.UserID(3)).
					Return(int64(0), cartErrors.ErrProductNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "insufficient stock",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					CheckoutCart(gomock.Any(), entity.UserID(3)).
					Return(int64(0), cartErrors.ErrInsufficientStock)
			},
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "service error",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					CheckoutCart(gomock.Any(), entity.UserID(3)).
					Return(int64(0), serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartService := mocks.NewMockcartService(ctrl)
			tt.setup(cartService)

			server := NewCartServer(nil, cartService, zap.NewNop())

			resp, err := server.CheckoutCart(context.Background(), &cartv1.CheckoutCartRequest{UserId: 3})
			require.Equal(t, tt.wantCode, status.Code(err))
			if tt.wantCode == codes.OK {
				require.Equal(t, tt.wantOrderID, resp.GetOrderId())
			}
		})
	}
}

func TestCartServer_ListCart(t *testing.T) {
	t.Parallel()

	items := []entity.ItemInfo{
		{
			Item:  entity.Item{SKU: 10, Count: 2},
			Name:  "keyboard",
			Price: 100,
		},
	}

	tests := []struct {
		name     string
		setup    func(*mocks.MockcartService)
		wantSent []*cartv1.ListCartResponse
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					ListCart(gomock.Any(), entity.UserID(3)).
					Return(items, uint32(200), nil)
			},
			wantSent: []*cartv1.ListCartResponse{
				{
					Items: []*cartv1.Item{
						{Sku: 10, Count: 2, Name: "keyboard", Price: 100},
					},
					TotalPrice: 200,
				},
			},
			wantCode: codes.OK,
		},
		{
			name: "empty cart",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					ListCart(gomock.Any(), entity.UserID(3)).
					Return([]entity.ItemInfo{}, uint32(0), nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "product not found",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					ListCart(gomock.Any(), entity.UserID(3)).
					Return(nil, uint32(0), cartErrors.ErrProductNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "service error",
			setup: func(cartService *mocks.MockcartService) {
				cartService.EXPECT().
					ListCart(gomock.Any(), entity.UserID(3)).
					Return(nil, uint32(0), serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartService := mocks.NewMockcartService(ctrl)
			tt.setup(cartService)

			server := NewCartServer(nil, cartService, zap.NewNop())
			stream := &listCartServer{ctx: context.Background()}

			err := server.ListCart(&cartv1.ListCartRequest{UserId: 3}, stream)
			require.Equal(t, tt.wantCode, status.Code(err))
			require.Equal(t, tt.wantSent, stream.sent)
		})
	}
}

type listCartServer struct {
	grpc.ServerStream
	ctx  context.Context
	sent []*cartv1.ListCartResponse
}

func (s *listCartServer) Context() context.Context {
	return s.ctx
}

func (s *listCartServer) Send(resp *cartv1.ListCartResponse) error {
	s.sent = append(s.sent, resp)
	return nil
}
