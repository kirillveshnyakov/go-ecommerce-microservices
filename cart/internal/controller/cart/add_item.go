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
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *cartServer) AddItem(ctx context.Context, req *cartv1.AddItemRequest) (*emptypb.Empty, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	if err := s.itemService.AddItem(ctx, entity.UserID(req.GetUserId()), entity.SKU(req.GetSku()), req.GetCount()); err != nil {
		switch {
		case errors.Is(err, cartErrors.ErrProductNotFound):
			return nil, status.Errorf(codes.NotFound, "product not found")
		case errors.Is(err, cartErrors.ErrInsufficientStock):
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient stock")
		default:
			s.logger.Error("cart controller - add item failed",
				zap.Int64("user_id", req.GetUserId()),
				zap.Uint32("sku", req.GetSku()),
				zap.Uint32("count", req.GetCount()),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	return &emptypb.Empty{}, nil
}
