package grpc

import (
	"context"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	productv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type productClient struct {
	client productv1.ProductServiceClient
}

func NewProductClient(conn grpclib.ClientConnInterface) *productClient {
	return &productClient{
		client: productv1.NewProductServiceClient(conn),
	}
}

func (c *productClient) GetProduct(ctx context.Context, sku entity.SKU) (port.ProductInfo, error) {
	resp, err := c.client.GetProduct(ctx, &productv1.GetProductRequest{Sku: uint32(sku)})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return port.ProductInfo{}, fmt.Errorf("product client - get product: sku=%d: %w", sku, cartErrors.ErrProductNotFound)
		}
		return port.ProductInfo{}, fmt.Errorf("product client - get product: sku=%d: %w", sku, err)
	}
	return port.ProductInfo{
		Name:  resp.GetName(),
		Price: resp.GetPrice(),
	}, nil
}
