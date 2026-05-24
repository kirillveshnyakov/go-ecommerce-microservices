package notifier

import (
	"context"
	"errors"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationsErrors "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/port"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/usecase/notifier/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestNotificationsService_SendNotification(t *testing.T) {
	t.Parallel()

	clientErr := errors.New("client error")

	tests := []struct {
		name    string
		setup   func(*mocks.Mockclient)
		wantErr error
	}{
		{
			name: "success",
			setup: func(client *mocks.Mockclient) {
				client.EXPECT().
					SendMessage(gomock.Any(), port.CallbackPayload{
						UserID:  42,
						OrderID: 777,
						Status:  "paid",
					}).
					Return(nil)
			},
		},
		{
			name: "empty callback address",
			setup: func(client *mocks.Mockclient) {
				client.EXPECT().
					SendMessage(gomock.Any(), port.CallbackPayload{
						UserID:  42,
						OrderID: 777,
						Status:  "paid",
					}).
					Return(notificationsErrors.ErrEmptyCallbackAddr)
			},
			wantErr: notificationsErrors.ErrEmptyCallbackAddr,
		},
		{
			name: "send notification failed",
			setup: func(client *mocks.Mockclient) {
				client.EXPECT().
					SendMessage(gomock.Any(), port.CallbackPayload{
						UserID:  42,
						OrderID: 777,
						Status:  "paid",
					}).
					Return(notificationsErrors.ErrSendNotification)
			},
			wantErr: notificationsErrors.ErrSendNotification,
		},
		{
			name: "client error",
			setup: func(client *mocks.Mockclient) {
				client.EXPECT().
					SendMessage(gomock.Any(), port.CallbackPayload{
						UserID:  42,
						OrderID: 777,
						Status:  "paid",
					}).
					Return(clientErr)
			},
			wantErr: clientErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			client := mocks.NewMockclient(ctrl)

			tt.setup(client)

			service := NewRestNotifier(client, zap.NewNop())

			err := service.SendOrderStatusChangeNotification(context.Background(), entity.Message{
				UserID:  42,
				OrderID: 777,
				Status:  entity.OrderStatusPaid,
			})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
