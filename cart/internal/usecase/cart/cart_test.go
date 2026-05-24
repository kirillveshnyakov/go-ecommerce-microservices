package cart

import (
	"context"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/usecase/cart/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestCartService_ListCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(*mocks.MockcartRepository, *mocks.MockproductClient)
		wantItems []entity.ItemInfo
		wantTotal uint32
		wantErr   error
	}{
		{
			name: "success",
			setup: func(cartRepository *mocks.MockcartRepository, productClient *mocks.MockproductClient) {
				items := []entity.Item{
					{SKU: 11, Count: 2},
					{SKU: 12, Count: 3},
				}

				cartRepository.EXPECT().
					GetCart(gomock.Any(), entity.UserID(35)).
					Return(items, nil)
				productClient.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(11)).
					Return(port.ProductInfo{Name: "keyboard", Price: 10}, nil)
				productClient.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(12)).
					Return(port.ProductInfo{Name: "mouse", Price: 7}, nil)
			},
			wantItems: []entity.ItemInfo{
				{
					Item:  entity.Item{SKU: 11, Count: 2},
					Name:  "keyboard",
					Price: 10,
				},
				{
					Item:  entity.Item{SKU: 12, Count: 3},
					Name:  "mouse",
					Price: 7,
				},
			},
			wantTotal: 41,
		},
		{
			name: "empty cart",
			setup: func(cartRepository *mocks.MockcartRepository, _ *mocks.MockproductClient) {
				cartRepository.EXPECT().
					GetCart(gomock.Any(), entity.UserID(35)).
					Return(nil, cartErrors.ErrUserCartNotFound)
			},
			wantItems: []entity.ItemInfo{},
			wantTotal: 0,
		},
		{
			name: "product not found",
			setup: func(cartRepository *mocks.MockcartRepository, productClient *mocks.MockproductClient) {
				cartRepository.EXPECT().
					GetCart(gomock.Any(), entity.UserID(35)).
					Return([]entity.Item{{SKU: 11, Count: 2}}, nil)
				productClient.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(11)).
					Return(port.ProductInfo{}, cartErrors.ErrProductNotFound)
			},
			wantErr: cartErrors.ErrProductNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepository := mocks.NewMockcartRepository(ctrl)
			productClient := mocks.NewMockproductClient(ctrl)

			tt.setup(cartRepository, productClient)

			service := NewCartService(cartRepository, productClient, nil, zap.NewNop(), nil)

			gotItems, gotTotal, err := service.ListCart(context.Background(), 35)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantItems, gotItems)
			require.Equal(t, tt.wantTotal, gotTotal)
		})
	}
}

func TestCartService_ClearCart(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	cartRepository := mocks.NewMockcartRepository(ctrl)

	cartRepository.EXPECT().
		ClearUserCart(gomock.Any(), entity.UserID(35)).
		Return(nil)

	service := NewCartService(cartRepository, nil, nil, zap.NewNop(), nil)

	require.NoError(t, service.ClearCart(context.Background(), 35))
}

func TestCartService_CheckoutCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(*mocks.MockcartRepository, *mocks.MocklomsClient, *mocks.Mocktransactor)
		wantOrderID int64
		wantErr     error
	}{
		{
			name: "success",
			setup: func(cartRepository *mocks.MockcartRepository, lomsClient *mocks.MocklomsClient, transactor *mocks.Mocktransactor) {
				items := []entity.Item{
					{SKU: 11, Count: 2},
					{SKU: 12, Count: 1},
				}

				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				cartRepository.EXPECT().
					GetCartWithLock(gomock.Any(), entity.UserID(35)).
					Return(items, nil)
				cartRepository.EXPECT().
					ClearUserCart(gomock.Any(), entity.UserID(35)).
					Return(nil)
				lomsClient.EXPECT().
					CreateOrder(gomock.Any(), port.CreateOrderRequest{
						UserID: 35,
						Items: []port.Item{
							{SKU: 11, Count: 2},
							{SKU: 12, Count: 1},
						},
					}).
					Return(int64(777), nil)
			},
			wantOrderID: 777,
		},
		{
			name: "loms insufficient stock",
			setup: func(cartRepository *mocks.MockcartRepository, lomsClient *mocks.MocklomsClient, transactor *mocks.Mocktransactor) {
				items := []entity.Item{{SKU: 11, Count: 2}}

				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				cartRepository.EXPECT().
					GetCartWithLock(gomock.Any(), entity.UserID(35)).
					Return(items, nil)
				cartRepository.EXPECT().
					ClearUserCart(gomock.Any(), entity.UserID(35)).
					Return(nil)
				lomsClient.EXPECT().
					CreateOrder(gomock.Any(), port.CreateOrderRequest{
						UserID: 35,
						Items:  []port.Item{{SKU: 11, Count: 2}},
					}).
					Return(int64(0), cartErrors.ErrInsufficientStock)
			},
			wantErr: cartErrors.ErrInsufficientStock,
		},
		{
			name: "loms product not found",
			setup: func(cartRepository *mocks.MockcartRepository, lomsClient *mocks.MocklomsClient, transactor *mocks.Mocktransactor) {
				items := []entity.Item{{SKU: 11, Count: 2}}

				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				cartRepository.EXPECT().
					GetCartWithLock(gomock.Any(), entity.UserID(35)).
					Return(items, nil)
				cartRepository.EXPECT().
					ClearUserCart(gomock.Any(), entity.UserID(35)).
					Return(nil)
				lomsClient.EXPECT().
					CreateOrder(gomock.Any(), port.CreateOrderRequest{
						UserID: 35,
						Items:  []port.Item{{SKU: 11, Count: 2}},
					}).
					Return(int64(0), cartErrors.ErrProductNotFound)
			},
			wantErr: cartErrors.ErrProductNotFound,
		},
		{
			name: "empty cart",
			setup: func(cartRepository *mocks.MockcartRepository, _ *mocks.MocklomsClient, transactor *mocks.Mocktransactor) {
				transactor.EXPECT().
					WithTx(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
						return f(ctx)
					})
				cartRepository.EXPECT().
					GetCartWithLock(gomock.Any(), entity.UserID(35)).
					Return(nil, cartErrors.ErrUserCartNotFound)
			},
			wantErr: cartErrors.ErrUserCartNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepository := mocks.NewMockcartRepository(ctrl)
			lomsClient := mocks.NewMocklomsClient(ctrl)
			transactor := mocks.NewMocktransactor(ctrl)

			tt.setup(cartRepository, lomsClient, transactor)

			service := NewCartService(cartRepository, nil, lomsClient, zap.NewNop(), transactor)

			gotOrderID, err := service.CheckoutCart(context.Background(), 35)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantOrderID, gotOrderID)
		})
	}
}
