package item

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	"go.uber.org/zap"
)

//go:generate mockgen -source=item.go -destination=mocks/item_mocks.go -package=mocks
type (
	cartRepository interface {
		AddItemChecked(ctx context.Context, userID entity.UserID, sku entity.SKU, count uint32, stock uint64) error
		DeleteItem(ctx context.Context, userID entity.UserID, sku entity.SKU) error
		GetItemCount(ctx context.Context, userID entity.UserID, sku entity.SKU) (uint32, error)
	}

	productClient interface {
		GetProduct(ctx context.Context, sku entity.SKU) (port.ProductInfo, error)
	}

	stocksClient interface {
		GetStock(ctx context.Context, sku entity.SKU) (uint64, error)
	}
)

type itemService struct {
	cartRepository cartRepository
	productClient  productClient
	stocksClient   stocksClient
	logger         *zap.Logger
}

func NewItemService(
	repository cartRepository,
	productClient productClient,
	stocksClient stocksClient,
	logger *zap.Logger,
) *itemService {
	return &itemService{
		cartRepository: repository,
		productClient:  productClient,
		stocksClient:   stocksClient,
		logger:         logger,
	}
}

func (s *itemService) AddItem(ctx context.Context, userID entity.UserID, sku entity.SKU, count uint32) error {
	_, err := s.productClient.GetProduct(ctx, sku)
	if err != nil {
		if errors.Is(err, cartErrors.ErrProductNotFound) {
			return err
		}
		return s.wrapAddItemError(err, userID, sku, count)
	}

	stock, err := s.stocksClient.GetStock(ctx, sku)
	if err != nil {
		if errors.Is(err, cartErrors.ErrProductNotFound) {
			return err
		}
		return s.wrapAddItemError(err, userID, sku, count)
	}

	if uint64(count) > stock {
		return fmt.Errorf(
			"item usecase - add item failed: user_id=%d sku=%d count=%d: %w",
			userID, sku, count, cartErrors.ErrInsufficientStock,
		)
	}

	err = s.cartRepository.AddItemChecked(ctx, userID, sku, count, stock)
	if err != nil {
		if errors.Is(err, cartErrors.ErrInsufficientStock) {
			return err
		}
		return s.wrapAddItemError(err, userID, sku, count)
	}
	return nil
}

func (s *itemService) DeleteItem(ctx context.Context, userID entity.UserID, sku entity.SKU) error {
	err := s.cartRepository.DeleteItem(ctx, userID, sku)
	if err != nil {
		return s.wrapDeleteItemError(err, userID, sku)
	}
	return nil
}

func (s *itemService) wrapAddItemError(err error, userID entity.UserID, sku entity.SKU, count uint32) error {
	s.logger.Error("item usecase - add item failed",
		zap.Int64("user_id", int64(userID)),
		zap.Uint32("sku", uint32(sku)),
		zap.Uint32("count", count),
		zap.Error(err),
	)

	return fmt.Errorf("item usecase - add item failed: user_id=%d sku=%d count=%d: %w", userID, sku, count, err)
}

func (s *itemService) wrapDeleteItemError(err error, userID entity.UserID, sku entity.SKU) error {
	s.logger.Error("item usecase - delete item failed",
		zap.Int64("user_id", int64(userID)),
		zap.Uint32("sku", uint32(sku)),
		zap.Error(err),
	)

	return fmt.Errorf("item usecase - delete item failed: user_id=%d sku=%d: %w", userID, sku, err)
}
