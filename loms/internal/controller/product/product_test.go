package product

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/product/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	productv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var serviceError = errors.New("service error")

func TestProductServer_CreateProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*mocks.MockproductService)
		wantSKU  uint32
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(productService *mocks.MockproductService) {
				productService.EXPECT().
					AddProduct(gomock.Any(), "keyboard", uint32(100)).
					Return(entity.SKU(10), nil)
			},
			wantSKU:  10,
			wantCode: codes.OK,
		},
		{
			name: "service error",
			setup: func(productService *mocks.MockproductService) {
				productService.EXPECT().
					AddProduct(gomock.Any(), "keyboard", uint32(100)).
					Return(entity.SKU(0), serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productService := mocks.NewMockproductService(ctrl)
			tt.setup(productService)

			server := NewProductServer(productService, zap.NewNop())

			resp, err := server.CreateProduct(context.Background(), &productv1.CreateProductRequest{
				Name:  "keyboard",
				Price: 100,
			})
			require.Equal(t, tt.wantCode, status.Code(err))
			if tt.wantCode == codes.OK {
				require.Equal(t, tt.wantSKU, resp.GetSku())
			}
		})
	}
}

func TestProductServer_GetProduct(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(*mocks.MockproductService)
		wantName  string
		wantPrice uint32
		wantCode  codes.Code
	}{
		{
			name: "success",
			setup: func(productService *mocks.MockproductService) {
				productService.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(10)).
					Return("keyboard", uint32(100), nil)
			},
			wantName:  "keyboard",
			wantPrice: 100,
			wantCode:  codes.OK,
		},
		{
			name: "not found",
			setup: func(productService *mocks.MockproductService) {
				productService.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(10)).
					Return("", uint32(0), lomsErrors.ErrProductNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "service error",
			setup: func(productService *mocks.MockproductService) {
				productService.EXPECT().
					GetProduct(gomock.Any(), entity.SKU(10)).
					Return("", uint32(0), serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			productService := mocks.NewMockproductService(ctrl)
			tt.setup(productService)

			server := NewProductServer(productService, zap.NewNop())

			resp, err := server.GetProduct(context.Background(), &productv1.GetProductRequest{Sku: 10})
			require.Equal(t, tt.wantCode, status.Code(err))
			if tt.wantCode == codes.OK {
				require.Equal(t, tt.wantName, resp.GetName())
				require.Equal(t, tt.wantPrice, resp.GetPrice())
			}
		})
	}
}
