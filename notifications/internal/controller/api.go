package controller

import (
	"context"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/controller/notifications"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type (
	notifier interface {
		SendOrderStatusChangeNotification(ctx context.Context, message entity.Message) error
	}
)

type API struct {
	notifications notificationsv1.NotificationsServer
}

func New(
	notifier notifier,
	logger *zap.Logger,
) *API {
	return &API{
		notifications: notifications.NewNotificationsServer(notifier, logger),
	}
}

func (a *API) RegisterGRPC(server *grpc.Server) {
	notificationsv1.RegisterNotificationsServer(server, a.notifications)
}
