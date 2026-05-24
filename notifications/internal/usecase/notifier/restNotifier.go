package notifier

import (
	"context"
	"errors"
	"fmt"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationsErrors "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/errors"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/port"
	"go.uber.org/zap"
)

//go:generate mockgen -source=restNotifier.go -destination=mocks/restNotifier_mocks.go -package=mocks
type (
	client interface {
		SendMessage(ctx context.Context, message port.CallbackPayload) error
	}
)

type notifier struct {
	client client
	logger *zap.Logger
}

func NewRestNotifier(client client, logger *zap.Logger) *notifier {
	return &notifier{
		client: client,
		logger: logger,
	}
}

func (s *notifier) SendOrderStatusChangeNotification(ctx context.Context, message entity.Message) error {
	if err := s.client.SendMessage(ctx, port.FromEntityToPortClientMessage(message)); err != nil {
		if errors.Is(err, notificationsErrors.ErrEmptyCallbackAddr) {
			return err
		}
		if errors.Is(err, notificationsErrors.ErrSendNotification) {
			return err
		}

		s.logger.Error("notifications usecase - send order status change notification failed",
			zap.Any("message", message),
			zap.Error(err),
		)

		return fmt.Errorf("notifications usecase - send order status change notification failed: message=%v: %w", message, err)
	}

	return nil
}
