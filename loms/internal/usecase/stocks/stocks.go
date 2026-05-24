package stocks

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"go.uber.org/zap"
)

//go:generate mockgen -source=stocks.go -destination=mocks/stocks_mocks.go -package=mocks
type (
	stocksRepository interface {
		GetStock(ctx context.Context, sku entity.SKU) (uint64, error)
		SetStock(ctx context.Context, sku entity.SKU, count uint64) error
	}
)

type stocksService struct {
	stocksRepository stocksRepository
	logger           *zap.Logger
}

func NewStocksService(stocksRepository stocksRepository, logger *zap.Logger) *stocksService {
	return &stocksService{
		stocksRepository: stocksRepository,
		logger:           logger,
	}
}

func (s *stocksService) GetStock(ctx context.Context, sku entity.SKU) (uint64, error) {
	stocks, err := s.stocksRepository.GetStock(ctx, sku)
	if err != nil {
		if errors.Is(err, lomsErrors.ErrProductNotFound) {
			return 0, err
		}

		return 0, s.wrapGetStockError(err, sku)
	}

	return stocks, nil
}

func (s *stocksService) SetStock(ctx context.Context, sku entity.SKU, count uint64) error {
	err := s.stocksRepository.SetStock(ctx, sku, count)
	if err != nil {
		if errors.Is(err, lomsErrors.ErrProductNotFound) {
			return err
		}

		return s.wrapSetStockError(err, sku, count)
	}

	return nil
}

func (s *stocksService) wrapGetStockError(err error, sku entity.SKU) error {
	s.logger.Error("stocks usecase - get stock failed",
		zap.Uint32("sku", uint32(sku)),
		zap.Error(err),
	)

	return fmt.Errorf("stocks usecase - get stock: sku=%d: %w", sku, err)
}

func (s *stocksService) wrapSetStockError(err error, sku entity.SKU, count uint64) error {
	s.logger.Error("stocks usecase - set stock failed",
		zap.Uint32("sku", uint32(sku)),
		zap.Uint64("count", count),
		zap.Error(err),
	)

	return fmt.Errorf("stocks usecase - set stock: sku=%d count=%d: %w", sku, count, err)
}
