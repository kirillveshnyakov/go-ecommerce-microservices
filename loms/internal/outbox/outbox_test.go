package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/outbox/mocks"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/port"
	repository "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/outbox"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestOutboxWorker_ProcessNotificationMessages(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	outboxRepository := mocks.NewMockoutboxRepository(ctrl)
	transactor := mocks.NewMocktransactor(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
			defer cancel()
			return f(ctx)
		}).
		AnyTimes()

	outboxRepository.EXPECT().
		GetMessages(gomock.Any(), 2, time.Minute).
		Return([]repository.Data{
			{
				IdempotencyKey: "777-awaiting_payment",
				Kind:           repository.KindNotification,
				Data: notificationBody(t, port.Notification{
					UserID:  42,
					OrderID: 777,
					Status:  port.OrderStatusAwaitingPayment,
				}),
			},
			{
				IdempotencyKey: "778-failed",
				Kind:           repository.KindNotification,
				Data: notificationBody(t, port.Notification{
					UserID:  42,
					OrderID: 778,
					Status:  port.OrderStatusFailed,
				}),
			},
		}, nil).
		AnyTimes()
	outboxRepository.EXPECT().
		MarkAsProcessed(gomock.Any(), []string{"777-awaiting_payment"}).
		Return(nil).
		AnyTimes()
	outboxRepository.EXPECT().
		MarkAsRetryable(gomock.Any(), []string{"778-failed"}).
		Return(nil).
		AnyTimes()

	outbox := New(zap.NewNop(), outboxRepository, notificationHandler, transactor)
	outbox.worker(ctx, 2, time.Millisecond, time.Minute)
}

func TestOutboxWorker_GetMessagesError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	outboxRepository := mocks.NewMockoutboxRepository(ctrl)
	transactor := mocks.NewMocktransactor(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	transactor.EXPECT().
		WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, f func(context.Context) error) error {
			defer cancel()
			return f(ctx)
		}).
		AnyTimes()
	outboxRepository.EXPECT().
		GetMessages(gomock.Any(), 2, time.Minute).
		Return(nil, errors.New("get messages error")).
		AnyTimes()

	outbox := New(zap.NewNop(), outboxRepository, notificationHandler, transactor)
	outbox.worker(ctx, 2, time.Millisecond, time.Minute)
}

func notificationHandler(kind repository.Kind) (KindHandler, error) {
	if kind != repository.KindNotification {
		return nil, errors.New("unsupported kind")
	}

	return func(_ context.Context, data []byte) error {
		var notification port.Notification
		if err := json.Unmarshal(data, &notification); err != nil {
			return err
		}
		if notification.Status == port.OrderStatusFailed {
			return errors.New("send notification error")
		}

		return nil
	}, nil
}

func notificationBody(t *testing.T, notification port.Notification) []byte {
	t.Helper()

	data, err := json.Marshal(notification)
	require.NoError(t, err)

	return data
}
