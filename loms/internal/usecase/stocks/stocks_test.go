package stocks

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usecase/stocks/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestStocksService_GetStock(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name      string
		setup     func(*mocks.MockstocksRepository)
		wantCount uint64
		wantErr   error
	}{
		{
			name: "success",
			setup: func(stocksRepository *mocks.MockstocksRepository) {
				stocksRepository.EXPECT().
					GetStock(gomock.Any(), entity.SKU(13)).
					Return(uint64(80), nil)
			},
			wantCount: 80,
		},
		{
			name: "product not found",
			setup: func(stocksRepository *mocks.MockstocksRepository) {
				stocksRepository.EXPECT().
					GetStock(gomock.Any(), entity.SKU(13)).
					Return(uint64(0), lomsErrors.ErrProductNotFound)
			},
			wantErr: lomsErrors.ErrProductNotFound,
		},
		{
			name: "repository error",
			setup: func(stocksRepository *mocks.MockstocksRepository) {
				stocksRepository.EXPECT().
					GetStock(gomock.Any(), entity.SKU(13)).
					Return(uint64(0), repositoryErr)
			},
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			stocksRepository := mocks.NewMockstocksRepository(ctrl)

			tt.setup(stocksRepository)

			service := NewStocksService(stocksRepository, zap.NewNop())

			count, err := service.GetStock(context.Background(), 13)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantCount, count)
		})
	}
}

func TestStocksService_SetStock(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name    string
		setup   func(*mocks.MockstocksRepository)
		wantErr error
	}{
		{
			name: "success",
			setup: func(stocksRepository *mocks.MockstocksRepository) {
				stocksRepository.EXPECT().
					SetStock(gomock.Any(), entity.SKU(13), uint64(80)).
					Return(nil)
			},
		},
		{
			name: "product not found",
			setup: func(stocksRepository *mocks.MockstocksRepository) {
				stocksRepository.EXPECT().
					SetStock(gomock.Any(), entity.SKU(13), uint64(80)).
					Return(lomsErrors.ErrProductNotFound)
			},
			wantErr: lomsErrors.ErrProductNotFound,
		},
		{
			name: "repository error",
			setup: func(stocksRepository *mocks.MockstocksRepository) {
				stocksRepository.EXPECT().
					SetStock(gomock.Any(), entity.SKU(13), uint64(80)).
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
			stocksRepository := mocks.NewMockstocksRepository(ctrl)

			tt.setup(stocksRepository)

			service := NewStocksService(stocksRepository, zap.NewNop())

			err := service.SetStock(context.Background(), 13, 80)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
