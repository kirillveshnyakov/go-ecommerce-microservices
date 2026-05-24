package cart

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
)

//go:generate mockgen -source=cart.go -destination=mocks/cart_mocks.go -package=mocks
type (
	itemService interface {
		AddItem(ctx context.Context, userID entity.UserID, sku entity.SKU, count uint32) error
		DeleteItem(ctx context.Context, userID entity.UserID, sku entity.SKU) error
	}

	cartService interface {
		ListCart(ctx context.Context, userID entity.UserID) ([]entity.ItemInfo, uint32, error)
		ClearCart(ctx context.Context, userID entity.UserID) error
		CheckoutCart(ctx context.Context, userID entity.UserID) (int64, error)
	}
)

type cartServer struct {
	itemService itemService
	cartService cartService
	logger      *zap.Logger
	cartv1.UnimplementedCartServer
}

func NewCartServer(
	itemService itemService,
	cartService cartService,
	logger *zap.Logger,
) *cartServer {
	return &cartServer{
		itemService: itemService,
		cartService: cartService,
		logger:      logger,
	}
}
