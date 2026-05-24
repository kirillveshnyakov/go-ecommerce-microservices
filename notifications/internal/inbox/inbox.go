package inbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/port"
	inboxRep "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/repository/inbox"
	"go.uber.org/zap"
)

//go:generate mockgen -source=inbox.go -destination=inbox_mocks.go -package=inbox
type (
	inboxRepository interface {
		GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration, maxAttempts int) ([]inboxRep.Data, error)
		MarkAsSuccess(ctx context.Context, idempotencyKeys []string) error
		MarkAsFailed(
			ctx context.Context,
			idempotencyKeys []string,
			errors []error,
			maxAttempts int,
			retryDelay time.Duration,
		) error
	}

	notifier interface {
		SendOrderStatusChangeNotification(ctx context.Context, message entity.Message) error
	}

	transactor interface {
		WithTx(ctx context.Context, f func(ctx context.Context) error) (err error)
	}
)

type Inbox interface {
	Start(ctx context.Context, cfg Config) func()
}

var _ Inbox = (*inboxImpl)(nil)

type inboxImpl struct {
	inboxRepository inboxRepository
	notifier        notifier
	transactor      transactor
	logger          *zap.Logger
}

func New(repository inboxRepository, notifier notifier, transactor transactor, logger *zap.Logger) *inboxImpl {
	return &inboxImpl{
		inboxRepository: repository,
		notifier:        notifier,
		transactor:      transactor,
		logger:          logger,
	}
}

type Config struct {
	Workers       int
	MaxAttempts   int
	BatchSize     int
	FetchPeriod   time.Duration
	RetryDelay    time.Duration
	InProgressTTL time.Duration
}

func (i *inboxImpl) Start(ctx context.Context, cfg Config) func() {
	wg := &sync.WaitGroup{}

	for workerID := 1; workerID <= cfg.Workers; workerID++ {
		wg.Go(func() {
			i.worker(ctx, cfg)
		})
	}

	return wg.Wait
}

func (i *inboxImpl) worker(ctx context.Context, cfg Config) {
	ticker := time.NewTicker(cfg.FetchPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := i.workerProcess(ctx, cfg); err != nil {
				i.logger.Error("inbox - worker error", zap.Error(err))
			}
		}
	}
}

func (i *inboxImpl) workerProcess(ctx context.Context, cfg Config) error {
	messages, err := i.inboxRepository.GetMessages(ctx, cfg.BatchSize, cfg.InProgressTTL, cfg.MaxAttempts)
	if err != nil {
		return fmt.Errorf("fetch messages: %w", err)
	}

	successKeys := make([]string, 0, len(messages)/2)
	failedKeys := make([]string, 0, len(messages)/2)
	processingErrors := make([]error, 0, len(messages)/2)

	for _, message := range messages {
		key := message.IdempotencyKey

		var kafkaMessage port.KafkaMessage
		if err = json.Unmarshal(message.Data, &kafkaMessage); err != nil {
			failedKeys = append(failedKeys, key)
			processingErrors = append(processingErrors, fmt.Errorf("unmarshal message: %w", err))
			continue
		}

		notification := port.FromPortToEntityKafkaMessage(kafkaMessage)

		if err = i.notifier.SendOrderStatusChangeNotification(ctx, notification); err != nil {
			failedKeys = append(failedKeys, key)
			processingErrors = append(processingErrors, err)
			continue
		}

		successKeys = append(successKeys, key)
	}

	var returnErr error
	for _, key := range successKeys {
		markErr := i.inboxRepository.MarkAsSuccess(ctx, []string{key})
		if markErr != nil {
			markErr = fmt.Errorf("mark as success: key=%s: %w", key, markErr)
			returnErr = errors.Join(returnErr, markErr)
		}
	}

	err = i.inboxRepository.MarkAsFailed(ctx, failedKeys, processingErrors, cfg.MaxAttempts, cfg.RetryDelay)
	if err != nil {
		err = fmt.Errorf("mark as failed: %w", err)
		returnErr = errors.Join(returnErr, err)
	}

	return returnErr
}
