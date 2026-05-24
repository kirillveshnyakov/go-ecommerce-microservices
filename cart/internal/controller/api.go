package controller

import (
	"context"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/controller/cart"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartsv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

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

type API struct {
	cart cartsv.CartServer
}

func New(itemService itemService, cartService cartService, logger *zap.Logger) *API {
	return &API{
		cart: cart.NewCartServer(itemService, cartService, logger),
	}
}

func (a *API) RegisterGRPC(server *grpc.Server) {
	cartsv.RegisterCartServer(server, a.cart)
}

func (a *API) RegisterGateway(ctx context.Context, mux *grpcruntime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return cartsv.RegisterCartHandlerFromEndpoint(ctx, mux, endpoint, opts)
}
