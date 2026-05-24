package port

import (
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	"github.com/stretchr/testify/require"
)

func TestFromEntityToPortStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   entity.OrderStatus
		want string
	}{
		{name: "new", in: entity.OrderStatusNew, want: StatusNew},
		{name: "awaiting payment", in: entity.OrderStatusAwaitingPayment, want: StatusAwaitingPayment},
		{name: "failed", in: entity.OrderStatusFailed, want: StatusFailed},
		{name: "paid", in: entity.OrderStatusPaid, want: StatusPaid},
		{name: "cancelled", in: entity.OrderStatusCancelled, want: StatusCancelled},
		{name: "unknown", in: entity.OrderStatus("unknown"), want: StatusUnavailable},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, FromEntityToPortStatus(tt.in))
		})
	}
}

func TestFromPortToEntityStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want entity.OrderStatus
	}{
		{name: "new", in: StatusNew, want: entity.OrderStatusNew},
		{name: "awaiting payment", in: StatusAwaitingPayment, want: entity.OrderStatusAwaitingPayment},
		{name: "failed", in: StatusFailed, want: entity.OrderStatusFailed},
		{name: "paid", in: StatusPaid, want: entity.OrderStatusPaid},
		{name: "cancelled", in: StatusCancelled, want: entity.OrderStatusCancelled},
		{name: "unknown", in: "unknown", want: entity.OrderStatusUnavailable},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, FromPortToEntityStatus(tt.in))
		})
	}
}

func TestClientMessageConverters(t *testing.T) {
	t.Parallel()

	entityMessage := entity.Message{
		UserID:  42,
		OrderID: 777,
		Status:  entity.OrderStatusPaid,
	}
	portMessage := CallbackPayload{
		UserID:  42,
		OrderID: 777,
		Status:  StatusPaid,
	}

	require.Equal(t, portMessage, FromEntityToPortClientMessage(entityMessage))
	require.Equal(t, entityMessage, FromPortToEntityClientMessage(portMessage))
}

func TestKafkaMessageConverters(t *testing.T) {
	t.Parallel()

	entityMessage := entity.Message{
		UserID:  42,
		OrderID: 777,
		Status:  entity.OrderStatusAwaitingPayment,
	}
	portMessage := KafkaMessage{
		UserID:  42,
		OrderID: 777,
		Status:  StatusAwaitingPayment,
	}

	require.Equal(t, portMessage, FromEntityToPortKafkaMessage(entityMessage))
	require.Equal(t, entityMessage, FromPortToEntityKafkaMessage(portMessage))
}
