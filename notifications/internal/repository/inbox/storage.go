package inbox

import (
	"context"
	"fmt"
	"time"

	sqlc "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/repository/inbox/sqlc"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5/pgtype"
)

type inboxRepository struct {
	queries *sqlc.Queries
}

func NewInboxRepository(db sqlc.DBTX) *inboxRepository {
	return &inboxRepository{
		queries: sqlc.New(db),
	}
}

func (r *inboxRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *inboxRepository) AddMessage(
	ctx context.Context,
	idempotencyKey string,
	message []byte,
	kafkaTopic string,
	kafkaPartition int32,
	kafkaOffset int64,
) error {
	err := r.getQueries(ctx).AddInboxMessage(ctx, sqlc.AddInboxMessageParams{
		IdempotencyKey: idempotencyKey,
		Data:           message,
		KafkaTopic:     kafkaTopic,
		KafkaPartition: kafkaPartition,
		KafkaOffset:    kafkaOffset,
	})
	if err != nil {
		return fmt.Errorf(
			"inbox repository - add message: key=%s message=%v kafka_topic=%s kafka_partition=%d kafka_offset=%d : %w",
			idempotencyKey, message, kafkaTopic, kafkaPartition, kafkaOffset, err,
		)
	}

	return nil
}

func (r *inboxRepository) AddDeadMessage(
	ctx context.Context,
	idempotencyKey string,
	message []byte,
	kafkaTopic string,
	kafkaPartition int32,
	kafkaOffset int64,
	messageErr error,
) error {
	lastErr := ""
	if messageErr != nil {
		lastErr = messageErr.Error()
	}

	err := r.getQueries(ctx).AddDeadInboxMessage(ctx, sqlc.AddDeadInboxMessageParams{
		IdempotencyKey: idempotencyKey,
		Data:           message,
		KafkaTopic:     kafkaTopic,
		KafkaPartition: kafkaPartition,
		KafkaOffset:    kafkaOffset,
		LastError:      lastErr,
	})
	if err != nil {
		return fmt.Errorf(
			"inbox repository - add dead message: key=%s message=%v kafka_topic=%s kafka_partition=%d kafka_offset=%d error=%s: %w",
			idempotencyKey, message, kafkaTopic, kafkaPartition, kafkaOffset, lastErr, err,
		)
	}

	return nil
}

type Data struct {
	IdempotencyKey string
	Data           []byte
}

func (r *inboxRepository) GetMessages(
	ctx context.Context,
	batchSize int,
	inProgressTTL time.Duration,
	maxAttempts int,
) ([]Data, error) {
	rows, err := r.getQueries(ctx).GetInboxMessages(ctx, sqlc.GetInboxMessagesParams{
		BatchSize: int32(batchSize),
		InProgressTtl: pgtype.Interval{
			Microseconds: inProgressTTL.Microseconds(),
			Valid:        true,
		},
		MaxAttempts: int32(maxAttempts),
	})
	if err != nil {
		return nil, fmt.Errorf("inbox repository - get messages: %w", err)
	}

	result := make([]Data, len(rows))
	for i, row := range rows {
		result[i] = Data{
			IdempotencyKey: row.IdempotencyKey,
			Data:           row.Data,
		}
	}

	return result, nil
}

func (r *inboxRepository) MarkAsSuccess(ctx context.Context, idempotencyKeys []string) error {
	if len(idempotencyKeys) == 0 {
		return nil
	}

	err := r.getQueries(ctx).MarkInboxMessagesAsSuccess(ctx, idempotencyKeys)
	if err != nil {
		return fmt.Errorf("inbox repository - mark inbox messages as success: keys=%s : %w", idempotencyKeys, err)
	}

	return nil
}

func (r *inboxRepository) MarkAsFailed(
	ctx context.Context,
	idempotencyKeys []string,
	errs []error,
	maxAttempts int,
	retryDelay time.Duration,
) error {
	if len(idempotencyKeys) == 0 {
		return nil
	}

	if len(idempotencyKeys) != len(errs) {
		return fmt.Errorf(
			"inbox repository - mark inbox messages as retryable: keys/errors length mismatch: keys=%d errors=%d",
			len(idempotencyKeys),
			len(errs),
		)
	}

	interval := pgtype.Interval{
		Microseconds: retryDelay.Microseconds(),
		Valid:        true,
	}

	q := r.getQueries(ctx)

	for i := 0; i < len(idempotencyKeys); i++ {
		lastErr := ""
		if errs[i] != nil {
			lastErr = errs[i].Error()
		}

		err := q.MarkInboxMessagesAsFailed(ctx, sqlc.MarkInboxMessagesAsFailedParams{
			IdempotencyKey: idempotencyKeys[i],
			LastError:      lastErr,
			MaxAttempts:    int32(maxAttempts),
			RetryDelay:     interval,
		})
		if err != nil {
			return fmt.Errorf(
				"inbox repository - mark inbox messages as failed: failed_key=%s keys=%s: %w",
				idempotencyKeys[i], idempotencyKeys, err,
			)
		}
	}

	return nil
}
