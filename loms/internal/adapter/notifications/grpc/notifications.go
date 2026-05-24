package grpc

import (
	"context"
	"fmt"

	lomsErrors "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type notificationsClient struct {
	client notificationsv1.NotificationsClient
}

func NewNotificationsClient(conn grpclib.ClientConnInterface) *notificationsClient {
	return &notificationsClient{
		client: notificationsv1.NewNotificationsClient(conn),
	}
}

func (c *notificationsClient) SendOrderStatusChangedNotification(ctx context.Context, message port.Notification) error {
	_, err := c.client.SendOrderStatusChangedNotification(ctx, &notificationsv1.OrderStatusChangedNotificationRequest{
		UserId:  message.UserID,
		OrderId: message.OrderID,
		Status:  port.FromPortToProtoStatus(message.Status),
	})
	if err != nil {
		if status.Code(err) == codes.Unavailable {
			return fmt.Errorf("notifications client - send notification: message=%v: %w", message, lomsErrors.ErrSendNotification)
		}
		return fmt.Errorf("notifications client - send notification: message=%v: %w", message, err)
	}
	return nil
}
