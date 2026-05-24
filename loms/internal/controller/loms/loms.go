package loms

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/converter"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	lomssv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate mockgen -source=loms.go -destination=mocks/loms_mocks.go -package=mocks
type (
	lomsService interface {
		CreateOrder(ctx context.Context, userID entity.UserID, items []entity.Item) (entity.OrderID, error)
		GetOrder(ctx context.Context, id entity.OrderID) (*entity.Order, error)
		PayOrder(ctx context.Context, id entity.OrderID) error
		CancelOrder(ctx context.Context, id entity.OrderID) error
	}
)

type lomsServer struct {
	lomsService lomsService
	logger      *zap.Logger
	lomssv.UnimplementedLomsServer
}

func NewLomsServer(lomsService lomsService, logger *zap.Logger) *lomsServer {
	return &lomsServer{
		lomsService: lomsService,
		logger:      logger,
	}
}

func (s *lomsServer) CreateOrder(ctx context.Context, req *lomssv.CreateOrderRequest) (*lomssv.CreateOrderResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	items := converter.ToEntityItemsArray(req.GetItems())

	orderID, err := s.lomsService.CreateOrder(ctx, entity.UserID(req.GetUserId()), items)
	if err != nil {
		switch {
		case errors.Is(err, lomsErrors.ErrInsufficientStock):
			return nil, status.Errorf(codes.FailedPrecondition, "reserve product error: insufficient stock")
		case errors.Is(err, lomsErrors.ErrProductNotFound):
			return nil, status.Errorf(codes.NotFound, "reserve product error: product not found")
		default:
			s.logger.Error("loms controller - create order failed",
				zap.Int64("user_id", req.GetUserId()),
				zap.String("items", fmt.Sprintf("%v", items)),
				zap.Error(err),
			)
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	s.logger.Info("loms controller - create order success",
		zap.Int64("user_id", req.GetUserId()),
		zap.Int64("order_id", int64(orderID)),
	)

	return &lomssv.CreateOrderResponse{
		OrderId: int64(orderID),
	}, nil
}

func (s *lomsServer) GetOrder(ctx context.Context, req *lomssv.GetOrderRequest) (*lomssv.GetOrderResponse, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	order, err := s.lomsService.GetOrder(ctx, entity.OrderID(req.GetOrderId()))
	if err != nil {
		if errors.Is(err, lomsErrors.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, "order not found")
		}

		s.logger.Error("loms controller - get order failed",
			zap.Int64("order_id", req.GetOrderId()),
			zap.Error(err),
		)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	return &lomssv.GetOrderResponse{
		Status:    converter.FromOrderStatus(order.Status),
		UserId:    int64(order.UserID),
		Items:     converter.FromEntityItemsArray(order.Items),
		CreatedAt: timestamppb.New(order.CreatedAt),
		UpdatedAt: timestamppb.New(order.UpdatedAt),
	}, nil
}

func (s *lomsServer) PayOrder(ctx context.Context, req *lomssv.PayOrderRequest) (*emptypb.Empty, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	err := s.lomsService.PayOrder(ctx, entity.OrderID(req.GetOrderId()))
	if err != nil {
		if errors.Is(err, lomsErrors.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, "order not found")
		}
		if errors.Is(err, lomsErrors.ErrInvalidStatus) {
			return nil, status.Errorf(codes.FailedPrecondition, "invalid order status for operation")
		}

		s.logger.Error("loms controller - pay order failed",
			zap.Int64("order_id", req.GetOrderId()),
			zap.Error(err),
		)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	s.logger.Info("loms controller - pay order success",
		zap.Int64("order_id", req.GetOrderId()),
	)

	return &emptypb.Empty{}, nil
}

func (s *lomsServer) CancelOrder(ctx context.Context, req *lomssv.CancelOrderRequest) (*emptypb.Empty, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	err := s.lomsService.CancelOrder(ctx, entity.OrderID(req.GetOrderId()))
	if err != nil {
		if errors.Is(err, lomsErrors.ErrOrderNotFound) {
			return nil, status.Errorf(codes.NotFound, "order not found")
		}
		if errors.Is(err, lomsErrors.ErrInvalidStatus) {
			return nil, status.Errorf(codes.FailedPrecondition, "invalid order status for operation")
		}
		if errors.Is(err, lomsErrors.ErrProductNotFound) {
			return nil, status.Errorf(codes.NotFound, "release product error: product not found")
		}
		if errors.Is(err, lomsErrors.ErrInsufficientStock) {
			return nil, status.Errorf(codes.ResourceExhausted, "release product error: insufficient stock")
		}

		s.logger.Error("loms controller - cancel order failed",
			zap.Int64("order_id", req.GetOrderId()),
			zap.Error(err),
		)

		return nil, status.Error(codes.Internal, "internal server error")
	}

	s.logger.Info("loms controller - cancel order success",
		zap.Int64("order_id", req.GetOrderId()),
	)

	return &emptypb.Empty{}, nil
}
