package product

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/product/sqlc"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5"
)

type productRepository struct {
	queries *sqlc.Queries
}

func NewProductRepository(db sqlc.DBTX) *productRepository {
	return &productRepository{
		queries: sqlc.New(db),
	}
}

func (r *productRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *productRepository) AddProduct(ctx context.Context, product entity.Product) (entity.SKU, error) {
	sku, err := r.getQueries(ctx).AddProduct(ctx, sqlc.AddProductParams{
		Name:  product.Name,
		Price: int64(product.Price),
	})
	if err != nil {
		return 0, fmt.Errorf("product repository - add product: name=%s price=%d: %w", product.Name, product.Price, err)
	}

	return entity.SKU(sku), nil
}

func (r *productRepository) GetProduct(ctx context.Context, sku entity.SKU) (entity.Product, error) {
	productInfo, err := r.getQueries(ctx).GetProductBySKU(ctx, int64(sku))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Product{}, fmt.Errorf("product repository - get product: sku=%d: %w", sku, lomsErrors.ErrProductNotFound)
		}
		return entity.Product{}, fmt.Errorf("product repository - get product: sku=%d: %w", sku, err)
	}
	return entity.Product{
		ID:    sku,
		Name:  productInfo.Name,
		Price: uint32(productInfo.Price),
	}, nil
}
