package inbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/port"
	inboxRep "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/repository/inbox"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestInboxWorkerProcess(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repository := NewMockinboxRepository(ctrl)
	notifier := NewMocknotifier(ctrl)

	notifyErr := errors.New("notify error")
	messages := []inboxRep.Data{
		{
			IdempotencyKey: "777-awaiting_payment",
			Data: inboxMessageBody(t, port.KafkaMessage{
				UserID:  42,
				OrderID: 777,
				Status:  port.StatusAwaitingPayment,
			}),
		},
		{
			IdempotencyKey: "778-paid",
			Data: inboxMessageBody(t, port.KafkaMessage{
				UserID:  42,
				OrderID: 778,
				Status:  port.StatusPaid,
			}),
		},
		{
			IdempotencyKey: "bad-json",
			Data:           []byte("{"),
		},
	}
	successNotification := entity.Message{
		UserID:  42,
		OrderID: 777,
		Status:  entity.OrderStatusAwaitingPayment,
	}
	failedNotification := entity.Message{
		UserID:  42,
		OrderID: 778,
		Status:  entity.OrderStatusPaid,
	}

	gomock.InOrder(
		repository.EXPECT().
			GetMessages(gomock.Any(), 3, time.Minute, 5).
			Return(messages, nil),
		notifier.EXPECT().
			SendOrderStatusChangeNotification(gomock.Any(), successNotification).
			Return(nil),
		notifier.EXPECT().
			SendOrderStatusChangeNotification(gomock.Any(), failedNotification).
			Return(notifyErr),
		repository.EXPECT().
			MarkAsSuccess(gomock.Any(), []string{"777-awaiting_payment"}).
			Return(nil),
		repository.EXPECT().
			MarkAsFailed(gomock.Any(), []string{"778-paid", "bad-json"}, gomock.Any(), 5, time.Second).
			DoAndReturn(func(_ context.Context, _ []string, processingErrors []error, _ int, _ time.Duration) error {
				require.Len(t, processingErrors, 2)
				require.ErrorIs(t, processingErrors[0], notifyErr)
				require.ErrorContains(t, processingErrors[1], "unmarshal message")
				return nil
			}),
	)

	inbox := New(repository, notifier, nil, zap.NewNop())
	err := inbox.workerProcess(context.Background(), Config{
		MaxAttempts:   5,
		BatchSize:     3,
		RetryDelay:    time.Second,
		InProgressTTL: time.Minute,
	})

	require.NoError(t, err)
}

func TestInboxWorkerProcessGetMessagesError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repository := NewMockinboxRepository(ctrl)
	notifier := NewMocknotifier(ctrl)

	getErr := errors.New("get messages error")

	repository.EXPECT().
		GetMessages(gomock.Any(), 3, time.Minute, 5).
		Return(nil, getErr)

	inbox := New(repository, notifier, nil, zap.NewNop())
	err := inbox.workerProcess(context.Background(), Config{
		MaxAttempts:   5,
		BatchSize:     3,
		RetryDelay:    time.Second,
		InProgressTTL: time.Minute,
	})

	require.ErrorIs(t, err, getErr)
}

func TestInboxWorkerProcessReturnsMarkErrors(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repository := NewMockinboxRepository(ctrl)
	notifier := NewMocknotifier(ctrl)

	markSuccessErr := errors.New("mark success error")
	markFailedErr := errors.New("mark failed error")
	notifyErr := errors.New("notify error")
	messages := []inboxRep.Data{
		{
			IdempotencyKey: "777-awaiting_payment",
			Data: inboxMessageBody(t, port.KafkaMessage{
				UserID:  42,
				OrderID: 777,
				Status:  port.StatusAwaitingPayment,
			}),
		},
		{
			IdempotencyKey: "778-paid",
			Data: inboxMessageBody(t, port.KafkaMessage{
				UserID:  42,
				OrderID: 778,
				Status:  port.StatusPaid,
			}),
		},
	}
	successNotification := entity.Message{
		UserID:  42,
		OrderID: 777,
		Status:  entity.OrderStatusAwaitingPayment,
	}
	failedNotification := entity.Message{
		UserID:  42,
		OrderID: 778,
		Status:  entity.OrderStatusPaid,
	}

	gomock.InOrder(
		repository.EXPECT().
			GetMessages(gomock.Any(), 2, time.Minute, 5).
			Return(messages, nil),
		notifier.EXPECT().
			SendOrderStatusChangeNotification(gomock.Any(), successNotification).
			Return(nil),
		notifier.EXPECT().
			SendOrderStatusChangeNotification(gomock.Any(), failedNotification).
			Return(notifyErr),
		repository.EXPECT().
			MarkAsSuccess(gomock.Any(), []string{"777-awaiting_payment"}).
			Return(markSuccessErr),
		repository.EXPECT().
			MarkAsFailed(gomock.Any(), []string{"778-paid"}, gomock.Any(), 5, time.Second).
			DoAndReturn(func(_ context.Context, _ []string, processingErrors []error, _ int, _ time.Duration) error {
				require.Len(t, processingErrors, 1)
				require.ErrorIs(t, processingErrors[0], notifyErr)
				return markFailedErr
			}),
	)

	inbox := New(repository, notifier, nil, zap.NewNop())
	err := inbox.workerProcess(context.Background(), Config{
		MaxAttempts:   5,
		BatchSize:     2,
		RetryDelay:    time.Second,
		InProgressTTL: time.Minute,
	})

	require.ErrorIs(t, err, markSuccessErr)
	require.ErrorIs(t, err, markFailedErr)
}

func TestInboxWorkerProcessNoMessages(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repository := NewMockinboxRepository(ctrl)
	notifier := NewMocknotifier(ctrl)

	gomock.InOrder(
		repository.EXPECT().
			GetMessages(gomock.Any(), 10, time.Minute, 5).
			Return(nil, nil),
		repository.EXPECT().
			MarkAsFailed(gomock.Any(), []string{}, []error{}, 5, time.Second).
			Return(nil),
	)

	inbox := New(repository, notifier, nil, zap.NewNop())
	err := inbox.workerProcess(context.Background(), Config{
		MaxAttempts:   5,
		BatchSize:     10,
		RetryDelay:    time.Second,
		InProgressTTL: time.Minute,
	})

	require.NoError(t, err)
}

func inboxMessageBody(t *testing.T, message port.KafkaMessage) []byte {
	t.Helper()

	data, err := json.Marshal(message)
	require.NoError(t, err)

	return data
}
