package outbox

import (
	"context"
	"fmt"
	"sync"
	"time"

	repository "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/outbox"
	"go.uber.org/zap"
)

//go:generate mockgen -source=outbox.go -destination=mocks/outbox_mocks.go -package=mocks
type (
	outboxRepository interface {
		GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]repository.Data, error)
		MarkAsProcessed(ctx context.Context, idempotencyKeys []string) error
		MarkAsRetryable(ctx context.Context, idempotencyKeys []string) error
	}

	transactor interface {
		WithTx(ctx context.Context, f func(ctx context.Context) error) (err error)
	}
)

type GlobalHandler = func(kind repository.Kind) (KindHandler, error)
type KindHandler = func(ctx context.Context, data []byte) error

type Outbox interface {
	Start(ctx context.Context, workers int, batchSize int, waitTime time.Duration, inProgressTTL time.Duration) func()
}

var _ Outbox = (*outboxImpl)(nil)

type outboxImpl struct {
	logger           *zap.Logger
	outboxRepository outboxRepository
	globalHandler    GlobalHandler
	transactor       transactor
}

func New(
	logger *zap.Logger,
	outboxRepository outboxRepository,
	globalHandler GlobalHandler,
	transactor transactor,
) *outboxImpl {
	return &outboxImpl{
		logger:           logger,
		outboxRepository: outboxRepository,
		globalHandler:    globalHandler,
		transactor:       transactor,
	}
}

func (o *outboxImpl) Start(
	ctx context.Context,
	workers int,
	batchSize int,
	fetchPeriod time.Duration,
	inProgressTTL time.Duration,
) func() {
	wg := &sync.WaitGroup{}

	for workerID := 1; workerID <= workers; workerID++ {
		wg.Go(func() {
			o.worker(ctx, batchSize, fetchPeriod, inProgressTTL)
		})
	}
	return wg.Wait
}

func (o *outboxImpl) worker(
	ctx context.Context,
	batchSize int,
	fetchPeriod time.Duration,
	inProgressTTL time.Duration,
) {
	ticker := time.NewTicker(fetchPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := o.transactor.WithTx(ctx, func(ctx context.Context) error {
				messages, err := o.outboxRepository.GetMessages(ctx, batchSize, inProgressTTL)

				if err != nil {
					o.logger.Error("outbox - fetch messages", zap.Error(err))
					return fmt.Errorf("outbox - fetch messages: %w", err)
				}

				successKeys := make([]string, 0, len(messages)/2)
				failedKeys := make([]string, 0, len(messages)/2)

				for i := 0; i < len(messages); i++ {
					message := messages[i]
					key := message.IdempotencyKey

					kindHandler, errGetHandler := o.globalHandler(message.Kind)

					if errGetHandler != nil {
						o.logger.Error("outbox - unexpected kind",
							zap.Error(errGetHandler),
							zap.Any("kind", message.Kind),
						)
						continue
					}

					err = kindHandler(ctx, message.Data)

					if err != nil {
						failedKeys = append(failedKeys, key)
						o.logger.Error("outbox - kind handler error",
							zap.Error(err),
							zap.Any("kind", message.Kind),
							zap.Any("message", message),
						)
						continue
					}

					successKeys = append(successKeys, key)
				}

				err = o.outboxRepository.MarkAsProcessed(ctx, successKeys)

				if err != nil {
					o.logger.Error("outbox - mark as processed outbox error", zap.Error(err))
					return fmt.Errorf("outbox - mark as processed outbox error: %w", err)
				}

				err = o.outboxRepository.MarkAsRetryable(ctx, failedKeys)

				if err != nil {
					o.logger.Error("outbox - mark as retryable error", zap.Error(err))
					return fmt.Errorf("outbox - mark as retryable error: %w", err)
				}

				return nil
			})

			if err != nil {
				o.logger.Error("outbox - worker transaction error", zap.Error(err))
			}
		}
	}
}
