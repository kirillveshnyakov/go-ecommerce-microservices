package grpc

import (
	"context"
	"fmt"

	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/port"
	lomsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type lomsClient struct {
	client lomsv1.LomsClient
}

func NewLOMSClient(conn grpclib.ClientConnInterface) *lomsClient {
	return &lomsClient{
		client: lomsv1.NewLomsClient(conn),
	}
}

func (c *lomsClient) CreateOrder(ctx context.Context, req port.CreateOrderRequest) (int64, error) {
	lomsItems := port.FromPortToLomsItemsArray(req.Items)
	resp, err := c.client.CreateOrder(ctx, &lomsv1.CreateOrderRequest{
		UserId: int64(req.UserID),
		Items:  lomsItems,
	})
	if err != nil {
		if status.Code(err) == codes.FailedPrecondition {
			return 0, fmt.Errorf("loms client - create order: user_id=%d item=%v: %w",
				req.UserID,
				req.Items,
				cartErrors.ErrInsufficientStock)
		}
		if status.Code(err) == codes.NotFound {
			return 0, fmt.Errorf("loms client - create order: user_id=%d item=%v: %w",
				req.UserID,
				req.Items,
				cartErrors.ErrProductNotFound)
		}
		return 0, fmt.Errorf("loms client - create order: user_id=%d item=%v: %w", req.UserID, req.Items, err)
	}
	return resp.GetOrderId(), nil
}
