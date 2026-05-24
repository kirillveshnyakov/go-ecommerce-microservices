package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"go.uber.org/zap"
)

//go:generate mockgen -source=product.go -destination=mocks/product_mocks.go -package=mocks
type (
	productRepository interface {
		GetProduct(ctx context.Context, sku entity.SKU) (entity.Product, error)
		AddProduct(ctx context.Context, product entity.Product) (entity.SKU, error)
	}
)

type productService struct {
	productRepository productRepository
	logger            *zap.Logger
}

func NewProductService(productRepository productRepository, logger *zap.Logger) *productService {
	return &productService{
		productRepository: productRepository,
		logger:            logger,
	}
}

func (s *productService) AddProduct(ctx context.Context, name string, price uint32) (entity.SKU, error) {
	sku, err := s.productRepository.AddProduct(ctx, entity.Product{
		Name:  name,
		Price: price,
	})
	if err != nil {
		return 0, s.wrapAddProductError(err, name, price)
	}

	return sku, nil
}

func (s *productService) GetProduct(ctx context.Context, sku entity.SKU) (string, uint32, error) {
	product, err := s.productRepository.GetProduct(ctx, sku)
	if err != nil {
		if errors.Is(err, lomsErrors.ErrProductNotFound) {
			return "", 0, err
		}

		return "", 0, s.wrapGetProductError(err, sku)
	}

	return product.Name, product.Price, nil
}

func (s *productService) wrapAddProductError(err error, name string, price uint32) error {
	s.logger.Error("product usecase - add product failed",
		zap.String("name", name),
		zap.Uint32("price", price),
		zap.Error(err),
	)

	return fmt.Errorf("product usecase - add product: name=%s price=%d: %w", name, price, err)
}

func (s *productService) wrapGetProductError(err error, sku entity.SKU) error {
	s.logger.Error("product usecase - get product failed",
		zap.Uint32("sku", uint32(sku)),
		zap.Error(err),
	)

	return fmt.Errorf("product usecase - get product: sku=%d: %w", sku, err)
}
