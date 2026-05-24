package cart

import (
	"errors"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/controller/converter"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartErrors "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/errors"
	cartv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *cartServer) ListCart(req *cartv1.ListCartRequest, srv cartv1.Cart_ListCartServer) error {
	if err := req.ValidateAll(); err != nil {
		return status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	items, totalPrice, err := s.cartService.ListCart(srv.Context(), entity.UserID(req.GetUserId()))
	if err != nil {
		switch {
		case errors.Is(err, cartErrors.ErrProductNotFound):
			s.logger.Error("cart controller - list cart failed",
				zap.Int64("user_id", req.GetUserId()),
				zap.Error(err),
			)
			return status.Errorf(codes.NotFound, "product in cart not found")
		default:
			s.logger.Error("cart controller - list cart failed",
				zap.Int64("user_id", req.GetUserId()),
				zap.Error(err),
			)
			return status.Error(codes.Internal, "internal server error")
		}
	}

	if len(items) > 0 {
		err = srv.Send(&cartv1.ListCartResponse{
			Items:      converter.FromEntityToCartItems(items),
			TotalPrice: totalPrice,
		})
		if err != nil {
			s.logger.Error("cart controller - list cart failed",
				zap.Int64("user_id", req.GetUserId()),
				zap.Error(err),
			)
			return status.Errorf(codes.Internal, "internal server error")
		}
	}

	return nil
}
