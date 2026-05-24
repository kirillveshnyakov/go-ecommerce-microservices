package item

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/usecase/item/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestItemService_AddItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(*mocks.MockcartRepository, *mocks.MockproductClient, *mocks.MockstocksClient)
		wantErr error
	}{
		{
			name: "success",
			setup: func(cartRepository *mocks.MockcartRepository, productClient *mocks.MockproductClient, stocksClient *mocks.MockstocksClient) {
				productClient.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(10)).
					Return(port.ProductInfo{Name: "keyboard", Price: 10}, nil)
				stocksClient.EXPECT().
					GetStock(gomock.Any(), entity.SKU(10)).
					Return(uint64(7), nil)
				cartRepository.EXPECT().
					AddItemChecked(gomock.Any(), entity.UserID(17), entity.SKU(10), uint32(2), uint64(7)).
					Return(nil)
			},
		},
		{
			name: "product not found",
			setup: func(_ *mocks.MockcartRepository, productClient *mocks.MockproductClient, _ *mocks.MockstocksClient) {
				productClient.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(10)).
					Return(port.ProductInfo{}, cartErrors.ErrProductNotFound)
			},
			wantErr: cartErrors.ErrProductNotFound,
		},
		{
			name: "stock not found",
			setup: func(_ *mocks.MockcartRepository, productClient *mocks.MockproductClient, stocksClient *mocks.MockstocksClient) {
				productClient.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(10)).
					Return(port.ProductInfo{Name: "keyboard", Price: 10}, nil)
				stocksClient.EXPECT().
					GetStock(gomock.Any(), entity.SKU(10)).
					Return(uint64(0), cartErrors.ErrProductNotFound)
			},
			wantErr: cartErrors.ErrProductNotFound,
		},
		{
			name: "insufficient stock",
			setup: func(_ *mocks.MockcartRepository, productClient *mocks.MockproductClient, stocksClient *mocks.MockstocksClient) {
				productClient.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(10)).
					Return(port.ProductInfo{Name: "keyboard", Price: 10}, nil)
				stocksClient.EXPECT().
					GetStock(gomock.Any(), entity.SKU(10)).
					Return(uint64(1), nil)
			},
			wantErr: cartErrors.ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			cartRepository := mocks.NewMockcartRepository(ctrl)
			productClient := mocks.NewMockproductClient(ctrl)
			stocksClient := mocks.NewMockstocksClient(ctrl)

			tt.setup(cartRepository, productClient, stocksClient)

			service := NewItemService(cartRepository, productClient, stocksClient, zap.NewNop())

			err := service.AddItem(context.Background(), 17, 10, 2)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestItemService_DeleteItem(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name    string
		setup   func(*mocks.MockcartRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(cartRepository *mocks.MockcartRepository) {
				cartRepository.EXPECT().
					DeleteItem(gomock.Any(), entity.UserID(33), entity.SKU(5)).
					Return(nil)
			},
		},
		{
			name: "repository error",
			setup: func(cartRepository *mocks.MockcartRepository) {
				cartRepository.EXPECT().
					DeleteItem(gomock.Any(), entity.UserID(33), entity.SKU(5)).
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
			cartRepository := mocks.NewMockcartRepository(ctrl)

			tt.setup(cartRepository)

			service := NewItemService(cartRepository, nil, nil, zap.NewNop())

			err := service.DeleteItem(context.Background(), 33, 5)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
