package product

import (
	"context"
	"errors"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	productsv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate mockgen -source=product.go -destination=mocks/product_mocks.go -package=mocks
type (
	productService interface {
		AddProduct(ctx context.Context, name string, price uint32) (entity.SKU, error)
		GetProduct(ctx context.Context, sku entity.SKU) (string, uint32, error)
	}
)

type productServer struct {
	productService productService
	logger         *zap.Logger
	productsv.UnimplementedProductServiceServer
}

func NewProductServer(productService productService, logger *zap.Logger) *productServer {
	return &productServer{
		productService: productService,
		logger:         logger,
	}
}

func (s *productServer) CreateProduct(ctx context.Context, req *productsv.CreateProductRequest) (*productsv.CreateProductResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	sku, err := s.productService.AddProduct(ctx, req.GetName(), req.GetPrice())

	if err != nil {
		s.logger.Error("product controller - create product failed",
			zap.String("name", req.GetName()),
			zap.Uint32("price", req.GetPrice()),
			zap.Error(err),
		)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	s.logger.Info("product controller - create product success",
		zap.String("name", req.GetName()),
		zap.Uint32("price", req.GetPrice()),
		zap.Uint32("sku", uint32(sku)),
	)

	return &productsv.CreateProductResponse{
		Sku: uint32(sku),
	}, nil
}

func (s *productServer) GetProduct(ctx context.Context, req *productsv.GetProductRequest) (*productsv.GetProductResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	name, price, err := s.productService.GetProduct(ctx, entity.SKU(req.GetSku()))
	if err != nil {
		if errors.Is(err, lomsErrors.ErrProductNotFound) {
			return nil, status.Errorf(codes.NotFound, "product not found")
		}

		s.logger.Error("product controller - get product failed",
			zap.Uint32("sku", req.GetSku()),
			zap.Error(err),
		)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &productsv.GetProductResponse{
		Name:  name,
		Price: price,
	}, nil
}
