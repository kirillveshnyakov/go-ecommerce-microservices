package outbox

import (
	"context"
	"fmt"
	"time"

	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/outbox/sqlc"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5/pgtype"
)

type outboxRepository struct {
	queries *sqlc.Queries
}

func NewOutboxRepository(db sqlc.DBTX) *outboxRepository {
	return &outboxRepository{
		queries: sqlc.New(db),
	}
}

func (r *outboxRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return r.queries.WithTx(tx)
	}

	return r.queries
}

func (r *outboxRepository) AddMessage(ctx context.Context, idempotencyKey string, kind Kind, message []byte) error {
	err := r.getQueries(ctx).AddOutboxMessage(ctx, sqlc.AddOutboxMessageParams{
		IdempotencyKey: idempotencyKey,
		Kind:           int32(kind),
		Data:           message,
	})
	if err != nil {
		return fmt.Errorf("outbox repository - add message: key=%s kind=%d message=%v: %w", idempotencyKey, kind, message, err)
	}

	return nil
}

func (r *outboxRepository) GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]Data, error) {
	rows, err := r.getQueries(ctx).GetOutboxMessages(ctx, sqlc.GetOutboxMessagesParams{
		BatchSize: int32(batchSize),
		InProgressTtl: pgtype.Interval{
			Microseconds: inProgressTTL.Microseconds(),
			Valid:        true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("outbox repository - get messages: %w", err)
	}

	result := make([]Data, len(rows))
	for i, row := range rows {
		result[i] = Data{
			IdempotencyKey: row.IdempotencyKey,
			Data:           row.Data,
			Kind:           Kind(row.Kind),
		}
	}

	return result, nil
}

func (r *outboxRepository) MarkAsProcessed(ctx context.Context, idempotencyKeys []string) error {
	if len(idempotencyKeys) == 0 {
		return nil
	}

	err := r.getQueries(ctx).MarkOutboxMessagesAsProcessed(ctx, idempotencyKeys)
	if err != nil {
		return fmt.Errorf("outbox repository - mark outbox messages as processed: keys=%s : %w", idempotencyKeys, err)
	}

	return nil
}

func (r *outboxRepository) MarkAsRetryable(ctx context.Context, idempotencyKeys []string) error {
	if len(idempotencyKeys) == 0 {
		return nil
	}

	err := r.getQueries(ctx).MarkOutboxMessagesAsRetryable(ctx, idempotencyKeys)
	if err != nil {
		return fmt.Errorf("outbox repository - mark outbox messages as retryable: keys=%s : %w", idempotencyKeys, err)
	}

	return nil
}
