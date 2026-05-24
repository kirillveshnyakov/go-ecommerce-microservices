package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	"go.uber.org/zap"
)

//go:generate mockgen -source=cart.go -destination=mocks/cart_mocks.go -package=mocks
type (
	cartRepository interface {
		ClearUserCart(ctx context.Context, userID entity.UserID) error
		GetCart(ctx context.Context, userID entity.UserID) ([]entity.Item, error)
		GetCartWithLock(ctx context.Context, userID entity.UserID) ([]entity.Item, error)
	}

	productClient interface {
		GetProduct(ctx context.Context, sku entity.SKU) (port.ProductInfo, error)
	}

	lomsClient interface {
		CreateOrder(ctx context.Context, req port.CreateOrderRequest) (int64, error)
	}

	transactor interface {
		WithTx(ctx context.Context, f func(ctx context.Context) error) (err error)
	}
)

type cartService struct {
	cartRepository cartRepository
	productClient  productClient
	lomsClient     lomsClient
	logger         *zap.Logger
	transactor     transactor
}

func NewCartService(
	repository cartRepository,
	productClient productClient,
	lomsClient lomsClient,
	logger *zap.Logger,
	transactor transactor,
) *cartService {
	return &cartService{
		cartRepository: repository,
		productClient:  productClient,
		lomsClient:     lomsClient,
		logger:         logger,
		transactor:     transactor,
	}
}

func (s *cartService) ListCart(ctx context.Context, userID entity.UserID) ([]entity.ItemInfo, uint32, error) {
	items, err := s.cartRepository.GetCart(ctx, userID)
	if err != nil {
		if errors.Is(err, cartErrors.ErrUserCartNotFound) {
			return make([]entity.ItemInfo, 0), 0, nil
		}
		return nil, 0, s.wrapListCartError(err, userID)
	}

	var totalPrice uint32
	itemsInfo := make([]entity.ItemInfo, len(items))

	for i, item := range items {
		product, err := s.productClient.GetProduct(ctx, item.SKU)
		if err != nil {
			if errors.Is(err, cartErrors.ErrProductNotFound) {
				return nil, 0, err
			}
			return nil, 0, s.wrapListCartError(err, userID)
		}
		totalPrice += product.Price * item.Count
		itemsInfo[i] = entity.ItemInfo{
			Item:  item,
			Name:  product.Name,
			Price: product.Price,
		}
	}

	return itemsInfo, totalPrice, nil
}

func (s *cartService) ClearCart(ctx context.Context, userID entity.UserID) error {
	if err := s.cartRepository.ClearUserCart(ctx, userID); err != nil {
		return s.wrapClearCartError(err, userID)
	}
	return nil
}

func (s *cartService) CheckoutCart(ctx context.Context, userID entity.UserID) (int64, error) {
	var orderID int64
	err := s.transactor.WithTx(ctx, func(ctx context.Context) error {
		items, err := s.cartRepository.GetCartWithLock(ctx, userID)
		if err != nil {
			if errors.Is(err, cartErrors.ErrUserCartNotFound) {
				return err
			}
			return s.wrapCheckoutCartError(err, userID)
		}

		err = s.cartRepository.ClearUserCart(ctx, userID)
		if err != nil {
			s.logger.Error("cart usecase - checkout cart - clear user cart failed",
				zap.Int64("user_id", int64(userID)),
				zap.Error(err),
			)
			return fmt.Errorf("cart usecase - list cart failed: user_id=%d: %w", userID, err)
		}

		orderID, err = s.lomsClient.CreateOrder(ctx, port.CreateOrderRequest{
			UserID: userID,
			Items:  port.FromEntityToPortItemsArray(items),
		})

		if err != nil {
			if errors.Is(err, cartErrors.ErrInsufficientStock) || errors.Is(err, cartErrors.ErrProductNotFound) {
				return err
			}
			return s.wrapCheckoutCartError(err, userID)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return orderID, nil
}

func (s *cartService) wrapListCartError(err error, userID entity.UserID) error {
	s.logger.Error("cart usecase - list cart failed",
		zap.Int64("user_id", int64(userID)),
		zap.Error(err),
	)

	return fmt.Errorf("cart usecase - list cart failed: user_id=%d: %w", userID, err)
}

func (s *cartService) wrapClearCartError(err error, userID entity.UserID) error {
	s.logger.Error("cart usecase - clear cart failed",
		zap.Int64("user_id", int64(userID)),
		zap.Error(err),
	)

	return fmt.Errorf("cart usecase - clear cart failed: user_id=%d: %w", userID, err)
}

func (s *cartService) wrapCheckoutCartError(err error, userID entity.UserID) error {
	s.logger.Error("cart usecase - checkout cart failed",
		zap.Int64("user_id", int64(userID)),
		zap.Error(err),
	)

	return fmt.Errorf("cart usecase - checkout cart failed: user_id=%d: %w", userID, err)
}
