package cart

import (
	"context"
	"errors"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	cartv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *cartServer) CheckoutCart(ctx context.Context, req *cartv1.CheckoutCartRequest) (*cartv1.CheckoutCartResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	orderID, err := s.cartService.CheckoutCart(ctx, entity.UserID(req.GetUserId()))
	if err != nil {
		switch {
		case errors.Is(err, cartErrors.ErrUserCartNotFound):
			return nil, status.Errorf(codes.NotFound, "user cart not found or empty")
		case errors.Is(err, cartErrors.ErrProductNotFound):
			return nil, status.Errorf(codes.NotFound, "product not found")
		case errors.Is(err, cartErrors.ErrInsufficientStock):
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient stock")
		default:
			s.logger.Error("cart controller - checkout cart failed",
				zap.Int64("user_id", req.GetUserId()),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	return &cartv1.CheckoutCartResponse{
		OrderId: orderID,
	}, nil
}
