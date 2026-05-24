package controller

import (
	"context"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	lomsController "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/loms"
	productController "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/product"
	stocksController "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/stocks"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomssv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	productsv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	stocksv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type (
	lomsService interface {
		CreateOrder(ctx context.Context, userID entity.UserID, items []entity.Item) (entity.OrderID, error)
		GetOrder(ctx context.Context, id entity.OrderID) (*entity.Order, error)
		PayOrder(ctx context.Context, id entity.OrderID) error
		CancelOrder(ctx context.Context, id entity.OrderID) error
	}

	productService interface {
		AddProduct(ctx context.Context, name string, price uint32) (entity.SKU, error)
		GetProduct(ctx context.Context, sku entity.SKU) (string, uint32, error)
	}

	stocksService interface {
		GetStock(ctx context.Context, sku entity.SKU) (uint64, error)
		SetStock(ctx context.Context, sku entity.SKU, count uint64) error
	}
)

type API struct {
	loms    lomssv.LomsServer
	product productsv.ProductServiceServer
	stocks  stocksv.StocksServer
}

func New(
	lomsService lomsService,
	productService productService,
	stocksService stocksService,
	logger *zap.Logger,
) *API {
	return &API{
		loms:    lomsController.NewLomsServer(lomsService, logger),
		product: productController.NewProductServer(productService, logger),
		stocks:  stocksController.NewStocksServer(stocksService, logger),
	}
}

func (a *API) RegisterGRPC(server *grpc.Server) {
	lomssv.RegisterLomsServer(server, a.loms)
	productsv.RegisterProductServiceServer(server, a.product)
	stocksv.RegisterStocksServer(server, a.stocks)
}

func (a *API) RegisterGateway(ctx context.Context, mux *grpcruntime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	if err := lomssv.RegisterLomsHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
		return err
	}
	if err := productsv.RegisterProductServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
		return err
	}
	if err := stocksv.RegisterStocksHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
		return err
	}
	return nil
}
