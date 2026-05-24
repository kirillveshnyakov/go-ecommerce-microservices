package stocks

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/stocks/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	stockv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var serviceError = errors.New("service error")

func TestStocksServer_GetStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(*mocks.MockstocksService)
		wantCount uint64
		wantCode  codes.Code
	}{
		{
			name: "success",
			setup: func(stocksService *mocks.MockstocksService) {
				stocksService.EXPECT().
					GetStock(gomock.Any(), entity.SKU(10)).
					Return(uint64(19), nil)
			},
			wantCount: 19,
			wantCode:  codes.OK,
		},
		{
			name: "not found",
			setup: func(stocksService *mocks.MockstocksService) {
				stocksService.EXPECT().
					GetStock(gomock.Any(), entity.SKU(10)).
					Return(uint64(0), lomsErrors.ErrProductNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "service error",
			setup: func(stocksService *mocks.MockstocksService) {
				stocksService.EXPECT().
					GetStock(gomock.Any(), entity.SKU(10)).
					Return(uint64(0), serviceError)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			stocksService := mocks.NewMockstocksService(ctrl)
			tt.setup(stocksService)

			server := NewStocksServer(stocksService, zap.NewNop())

			resp, err := server.GetStock(context.Background(), &stockv1.GetStockRequest{Sku: 10})
			require.Equal(t, tt.wantCode, status.Code(err))
			if tt.wantCode == codes.OK {
				require.Equal(t, tt.wantCount, resp.GetCount())
			}
		})
	}
}

func TestStocksServer_SetStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(*mocks.MockstocksService)
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(stocksService *mocks.MockstocksService) {
				stocksService.EXPECT().
					SetStock(gomock.Any(), entity.SKU(10), uint64(19)).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "not found",
			setup: func(stocksService *mocks.MockstocksService) {
				stocksService.EXPECT().
					SetStock(gomock.Any(), entity.SKU(10), uint64(19)).
					Return(lomsErrors.ErrProductNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "service error",
			setup: func(stocksService *mocks.MockstocksService) {
				stocksService.EXPECT().
					SetStock(gomock.Any(), entity.SKU(10), uint64(19)).
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
			stocksService := mocks.NewMockstocksService(ctrl)
			tt.setup(stocksService)

			server := NewStocksServer(stocksService, zap.NewNop())

			_, err := server.SetStock(context.Background(), &stockv1.SetStockRequest{
				Sku:   10,
				Count: 19,
			})
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}
