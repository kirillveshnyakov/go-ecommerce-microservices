package grpc

import (
	"context"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	stocksv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stocksClient struct {
	client stocksv1.StocksClient
}

func NewStocksClient(conn grpclib.ClientConnInterface) *stocksClient {
	return &stocksClient{
		client: stocksv1.NewStocksClient(conn),
	}
}

func (c *stocksClient) GetStock(ctx context.Context, sku entity.SKU) (uint64, error) {
	resp, err := c.client.GetStock(ctx, &stocksv1.GetStockRequest{Sku: uint32(sku)})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, fmt.Errorf("stocks client - get stock: sku=%d: %w", sku, cartErrors.ErrProductNotFound)
		}
		return 0, fmt.Errorf("stocks client - get stock: sku=%d: %w", sku, err)
	}
	return resp.GetCount(), nil
}
