package cart

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *cartServer) ClearCart(ctx context.Context, req *cartv1.ClearCartRequest) (*emptypb.Empty, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	if err := s.cartService.ClearCart(ctx, entity.UserID(req.GetUserId())); err != nil {
		s.logger.Error("cart controller - clear cart failed",
			zap.Int64("user_id", req.GetUserId()),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &emptypb.Empty{}, nil
}
