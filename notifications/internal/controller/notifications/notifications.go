package notifications

import (
	"context"
	"errors"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/controller/converter"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationsErrors "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/errors"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -source=notifications.go -destination=mocks/notifications_mocks.go -package=mocks
type (
	notifier interface {
		SendOrderStatusChangeNotification(ctx context.Context, message entity.Message) error
	}
)

type notificationsServer struct {
	notificationsService notifier
	logger               *zap.Logger
	notificationsv1.UnimplementedNotificationsServer
}

func NewNotificationsServer(
	notificationsService notifier,
	logger *zap.Logger,
) *notificationsServer {
	return &notificationsServer{
		notificationsService: notificationsService,
		logger:               logger,
	}
}

func (s *notificationsServer) SendOrderStatusChangedNotification(ctx context.Context, req *notificationsv1.OrderStatusChangedNotificationRequest) (*emptypb.Empty, error) {
	if err := req.ValidateAll(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation: %v", err)
	}

	if err := s.notificationsService.SendOrderStatusChangeNotification(ctx, converter.ToEntityMessage(
		req.GetUserId(), req.GetOrderId(), req.GetStatus(),
	)); err != nil {
		switch {
		case errors.Is(err, notificationsErrors.ErrEmptyCallbackAddr):
			return &emptypb.Empty{}, nil
		case errors.Is(err, notificationsErrors.ErrSendNotification):
			return nil, status.Error(codes.Unavailable, "failed to send notification")
		default:
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	return &emptypb.Empty{}, nil
}
