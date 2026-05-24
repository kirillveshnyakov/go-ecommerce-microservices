package kafka

import (
	"context"
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/adapter/kafka/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestCreateIdempotencyKey(t *testing.T) {
	t.Parallel()

	require.Equal(t, "777-awaiting_payment", createIdempotencyKey(777, "awaiting_payment"))
}

func TestCreateIdempotencyKeyForDead(t *testing.T) {
	t.Parallel()

	require.Equal(t, "kafka:order_status_notifications:2:15", createIdempotencyKeyForDead("order_status_notifications", 2, 15))
}

func TestJoinBrokers(t *testing.T) {
	t.Parallel()

	require.Equal(t, "kafka-1:9092,kafka-2:9092", joinBrokers([]string{"kafka-1:9092", "kafka-2:9092"}))
}

func TestRunConsumerNoBrokers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repository := mocks.NewMockinboxRepository(ctrl)

	err := RunConsumer(
		context.Background(),
		nil,
		"order_status_notifications",
		"notifications",
		repository,
		zap.NewNop(),
	)

	require.ErrorContains(t, err, "no kafka brokers")
}
