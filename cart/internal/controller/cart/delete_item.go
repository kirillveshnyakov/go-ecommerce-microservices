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

func (s *cartServer) DeleteItem(ctx context.Context, req *cartv1.DeleteItemRequest) (*emptypb.Empty, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	err := s.itemService.DeleteItem(ctx, entity.UserID(req.GetUserId()), entity.SKU(req.GetSku()))

	if err != nil {
		s.logger.Error("cart controller - delete item failed",
			zap.Int64("user_id", req.GetUserId()),
			zap.Uint32("sku", req.GetSku()),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &emptypb.Empty{}, nil
}
