package product

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usecase/product/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestProductService_AddProduct(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name    string
		setup   func(*mocks.MockproductRepository)
		wantSKU entity.SKU
		wantErr error
	}{
		{
			name: "success",
			setup: func(productRepository *mocks.MockproductRepository) {
				productRepository.EXPECT().
					AddProduct(gomock.Any(), entity.Product{Name: "keyboard", Price: 10}).
					Return(entity.SKU(4), nil)
			},
			wantSKU: 4,
		},
		{
			name: "repository error",
			setup: func(productRepository *mocks.MockproductRepository) {
				productRepository.EXPECT().
					AddProduct(gomock.Any(), entity.Product{Name: "keyboard", Price: 10}).
					Return(entity.SKU(0), repositoryErr)
			},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productRepository := mocks.NewMockproductRepository(ctrl)

			tt.setup(productRepository)

			service := NewProductService(productRepository, zap.NewNop())

			sku, err := service.AddProduct(context.Background(), "keyboard", 10)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSKU, sku)
		})
	}
}

func TestProductService_GetProduct(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name      string
		setup     func(*mocks.MockproductRepository)
		wantName  string
		wantPrice uint32
		wantErr   error
	}{
		{
			name: "success",
			setup: func(productRepository *mocks.MockproductRepository) {
				productRepository.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(4)).
					Return(entity.Product{ID: 4, Name: "keyboard", Price: 10}, nil)
			},
			wantName:  "keyboard",
			wantPrice: 10,
		},
		{
			name: "not found",
			setup: func(productRepository *mocks.MockproductRepository) {
				productRepository.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(4)).
					Return(entity.Product{}, lomsErrors.ErrProductNotFound)
			},
			wantErr: lomsErrors.ErrProductNotFound,
		},
		{
			name: "repository error",
			setup: func(productRepository *mocks.MockproductRepository) {
				productRepository.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(4)).
					Return(entity.Product{}, repositoryErr)
			},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productRepository := mocks.NewMockproductRepository(ctrl)

			tt.setup(productRepository)

			service := NewProductService(productRepository, zap.NewNop())

			name, price, err := service.GetProduct(context.Background(), 4)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, name)
			require.Equal(t, tt.wantPrice, price)
		})
	}
}
