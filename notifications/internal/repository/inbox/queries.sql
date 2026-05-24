-- name: AddInboxMessage :exec
INSERT INTO notifications.inbox (idempotency_key, data, kafka_topic, kafka_partition, kafka_offset)
VALUES (
           sqlc.arg(idempotency_key),
           sqlc.arg(data),
           sqlc.arg(kafka_topic),
           sqlc.arg(kafka_partition),
           sqlc.arg(kafka_offset)
       )
ON CONFLICT DO NOTHING;

-- name: AddDeadInboxMessage :exec
INSERT INTO notifications.inbox (
    idempotency_key,
    data,
    status,
    kafka_topic,
    kafka_partition,
    kafka_offset,
    attempts,
    last_error,
    dead_at
)
VALUES (
           sqlc.arg(idempotency_key),
           sqlc.arg(data),
           'DEAD'::inbox_status,
           sqlc.arg(kafka_topic),
           sqlc.arg(kafka_partition),
           sqlc.arg(kafka_offset),
           0,
           sqlc.arg(last_error)::text,
           now()
       )
ON CONFLICT DO NOTHING;

-- name: GetInboxMessages :many
WITH picked AS (
    SELECT i.idempotency_key
    FROM notifications.inbox AS i
    WHERE
        i.attempts < sqlc.arg(max_attempts)
      AND (
        i.status = 'CREATED'::inbox_status
            OR (
                i.status = 'RETRYABLE'::inbox_status
                AND i.next_retry_at <= now()
            )
            OR (
                i.status = 'IN_PROGRESS'::inbox_status
                AND i.updated_at < now() - sqlc.arg(in_progress_ttl)::interval
            )
        )
    ORDER BY i.created_at
    LIMIT sqlc.arg(batch_size)
    FOR UPDATE SKIP LOCKED
            )
UPDATE notifications.inbox AS i
SET
    status = 'IN_PROGRESS'::inbox_status,
    attempts = i.attempts + 1
FROM picked
WHERE i.idempotency_key = picked.idempotency_key
    RETURNING i.idempotency_key, i.data;

-- name: MarkInboxMessagesAsSuccess :exec
UPDATE notifications.inbox
SET status = 'SUCCESS'::inbox_status
WHERE idempotency_key = ANY (sqlc.arg(idempotency_keys)::text[])
    AND status = 'IN_PROGRESS'::inbox_status;

-- name: MarkInboxMessagesAsFailed :exec
UPDATE notifications.inbox
SET
    status = CASE
        WHEN attempts >= sqlc.arg(max_attempts)
            THEN 'DEAD'::inbox_status
        ELSE 'RETRYABLE'::inbox_status
    END,
    last_error = sqlc.arg(last_error)::text,
    dead_at = CASE
        WHEN attempts >= sqlc.arg(max_attempts)
            THEN now()
        ELSE dead_at
    END,
    next_retry_at = CASE
        WHEN attempts >= sqlc.arg(max_attempts)
            THEN next_retry_at
        ELSE now() + sqlc.arg(retry_delay)::interval
    END
WHERE idempotency_key = sqlc.arg(idempotency_key)::text
    AND status = 'IN_PROGRESS'::inbox_status;