-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS notifications;

CREATE TYPE inbox_status as ENUM ('CREATED', 'IN_PROGRESS', 'RETRYABLE', 'SUCCESS', 'DEAD');

CREATE TABLE notifications.inbox
(
    idempotency_key TEXT PRIMARY KEY,
    data            JSONB                          NOT NULL,
    status          inbox_status DEFAULT 'CREATED' NOT NULL,
    kafka_topic     TEXT                           NOT NULL,
    kafka_partition INT                            NOT NULL,
    kafka_offset    BIGINT                         NOT NULL,
    attempts        INT          DEFAULT 0         NOT NULL,
    last_error      TEXT,
    dead_at         TIMESTAMP,
    next_retry_at   TIMESTAMP    DEFAULT NOW()     NOT NULL,
    created_at      TIMESTAMP    DEFAULT NOW()     NOT NULL,
    updated_at      TIMESTAMP    DEFAULT NOW()     NOT NULL
);

CREATE INDEX idx_inbox_fetch ON notifications.inbox (status, next_retry_at, updated_at);

CREATE UNIQUE INDEX idx_inbox_kafka_message ON notifications.inbox (kafka_topic, kafka_partition, kafka_offset);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_inbox_timestamp() RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = now();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE OR REPLACE TRIGGER trigger_update_inbox_timestamp
    BEFORE UPDATE
    ON notifications.inbox
    FOR EACH ROW
EXECUTE FUNCTION update_inbox_timestamp();

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trigger_update_inbox_timestamp ON notifications.inbox;
DROP FUNCTION IF EXISTS update_inbox_timestamp;
DROP INDEX IF EXISTS notifications.idx_inbox_kafka_message;
DROP INDEX IF EXISTS notifications.idx_inbox_fetch;
DROP TABLE IF EXISTS notifications.inbox;
DROP TYPE IF EXISTS inbox_status;
-- +goose StatementEnd