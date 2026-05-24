package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestKafkaBrokerAddrs(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	cfg.Kafka.Brokers = " kafka-1:9092, ,kafka-2:9092,, kafka-3:9092 "

	require.Equal(t, []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"}, cfg.KafkaBrokerAddrs())
}

func TestConstructPostgresURL(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	cfg.PG.User = "user"
	cfg.PG.Password = "pass"
	cfg.PG.Host = "postgres"
	cfg.PG.Port = "5433"
	cfg.PG.DB = "db"

	require.Equal(t, "postgres://user:pass@postgres:5433/db?sslmode=disable", cfg.ConstructPostgresURL())
}

func TestNewDefaultConfig(t *testing.T) {
	t.Setenv("CALLBACK_ADDR", "")
	t.Setenv("GRPC_PORT", "")
	t.Setenv("GRPC_SHUTDOWN_TIME", "")
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("KAFKA_NOTIFICATIONS_TOPIC", "")
	t.Setenv("KAFKA_CONSUMER_GROUP", "")
	t.Setenv("INBOX_WORKERS", "")
	t.Setenv("INBOX_MAX_ATTEMPTS", "")
	t.Setenv("INBOX_BATCH_SIZE", "")
	t.Setenv("INBOX_FETCH_PERIOD", "")
	t.Setenv("INBOX_RETRY_DELAY", "")
	t.Setenv("INBOX_IN_PROGRESS_TTL", "")
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")

	cfg, err := New()
	require.NoError(t, err)

	require.Equal(t, "50053", cfg.GRPC.Port)
	require.Equal(t, 5*time.Second, cfg.GRPC.GrpcShutdownTime)
	require.Equal(t, "localhost:9092", cfg.Kafka.Brokers)
	require.Equal(t, "order_status_notifications", cfg.Kafka.Topic)
	require.Equal(t, "notifications", cfg.Kafka.ConsumerGroup)
	require.Equal(t, 2, cfg.Inbox.Workers)
	require.Equal(t, 20, cfg.Inbox.MaxAttempts)
	require.Equal(t, 5, cfg.Inbox.BatchSize)
	require.Equal(t, 200*time.Millisecond, cfg.Inbox.FetchPeriod)
	require.Equal(t, time.Second, cfg.Inbox.RetryDelay)
	require.Equal(t, 30*time.Second, cfg.Inbox.InProgressTTL)
	require.Equal(t, "localhost", cfg.PG.Host)
	require.Equal(t, "5432", cfg.PG.Port)
	require.Equal(t, "ecommerce", cfg.PG.DB)
}

func TestNewConfigFromEnv(t *testing.T) {
	t.Setenv("CALLBACK_ADDR", "callback:8080")
	t.Setenv("GRPC_PORT", "50099")
	t.Setenv("GRPC_SHUTDOWN_TIME", "3s")
	t.Setenv("KAFKA_BROKERS", "kafka-1:9092,kafka-2:9092")
	t.Setenv("KAFKA_NOTIFICATIONS_TOPIC", "topic")
	t.Setenv("KAFKA_CONSUMER_GROUP", "group")
	t.Setenv("INBOX_WORKERS", "4")
	t.Setenv("INBOX_MAX_ATTEMPTS", "11")
	t.Setenv("INBOX_BATCH_SIZE", "9")
	t.Setenv("INBOX_FETCH_PERIOD", "150ms")
	t.Setenv("INBOX_RETRY_DELAY", "2s")
	t.Setenv("INBOX_IN_PROGRESS_TTL", "45s")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "15432")
	t.Setenv("POSTGRES_DB", "custom_db")
	t.Setenv("POSTGRES_USER", "custom_user")
	t.Setenv("POSTGRES_PASSWORD", "custom_pass")

	cfg, err := New()
	require.NoError(t, err)

	require.Equal(t, "callback:8080", cfg.Clients.CallbackAddr)
	require.Equal(t, "50099", cfg.GRPC.Port)
	require.Equal(t, 3*time.Second, cfg.GRPC.GrpcShutdownTime)
	require.Equal(t, []string{"kafka-1:9092", "kafka-2:9092"}, cfg.KafkaBrokerAddrs())
	require.Equal(t, "topic", cfg.Kafka.Topic)
	require.Equal(t, "group", cfg.Kafka.ConsumerGroup)
	require.Equal(t, 4, cfg.Inbox.Workers)
	require.Equal(t, 11, cfg.Inbox.MaxAttempts)
	require.Equal(t, 9, cfg.Inbox.BatchSize)
	require.Equal(t, 150*time.Millisecond, cfg.Inbox.FetchPeriod)
	require.Equal(t, 2*time.Second, cfg.Inbox.RetryDelay)
	require.Equal(t, 45*time.Second, cfg.Inbox.InProgressTTL)
	require.Equal(t, "postgres://custom_user:custom_pass@postgres:15432/custom_db?sslmode=disable", cfg.ConstructPostgresURL())
}
