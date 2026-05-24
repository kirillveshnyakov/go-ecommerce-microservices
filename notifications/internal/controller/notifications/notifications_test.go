package notifications

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/controller/notifications/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationsErrors "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/errors"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNotificationsServer_SendOrderStatusChangedNotification(t *testing.T) {
	t.Parallel()

	serviceErr := errors.New("service error")

	message := entity.Message{
		UserID:  12,
		OrderID: 3,
		Status:  entity.OrderStatusPaid,
	}

	tests := []struct {
		name     string
		setup    func(*mocks.Mocknotifier)
		wantCode codes.Code
	}{
		{
			name: "success",
			setup: func(notifier *mocks.Mocknotifier) {
				notifier.EXPECT().
					SendOrderStatusChangeNotification(gomock.Any(), message).
					Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "empty callback address",
			setup: func(notifier *mocks.Mocknotifier) {
				notifier.EXPECT().
					SendOrderStatusChangeNotification(gomock.Any(), message).
					Return(notificationsErrors.ErrEmptyCallbackAddr)
			},
			wantCode: codes.OK,
		},
		{
			name: "send failed",
			setup: func(notifier *mocks.Mocknotifier) {
				notifier.EXPECT().
					SendOrderStatusChangeNotification(gomock.Any(), message).
					Return(notificationsErrors.ErrSendNotification)
			},
			wantCode: codes.Unavailable,
		},
		{
			name: "service error",
			setup: func(notifier *mocks.Mocknotifier) {
				notifier.EXPECT().
					SendOrderStatusChangeNotification(gomock.Any(), message).
					Return(serviceErr)
			},
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			notifier := mocks.NewMocknotifier(ctrl)
			tt.setup(notifier)

			server := NewNotificationsServer(notifier, zap.NewNop())

			_, err := server.SendOrderStatusChangedNotification(context.Background(), &notificationsv1.OrderStatusChangedNotificationRequest{
				UserId:  12,
				OrderId: 3,
				Status:  notificationsv1.OrderStatus_ORDER_STATUS_PAID,
			})
			require.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestNotificationsServer_SendOrderStatusChangedNotificationValidationError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	notifier := mocks.NewMocknotifier(ctrl)

	server := NewNotificationsServer(notifier, zap.NewNop())

	_, err := server.SendOrderStatusChangedNotification(context.Background(), &notificationsv1.OrderStatusChangedNotificationRequest{
		UserId:  -1,
		OrderId: 3,
		Status:  notificationsv1.OrderStatus_ORDER_STATUS_PAID,
	})

	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
