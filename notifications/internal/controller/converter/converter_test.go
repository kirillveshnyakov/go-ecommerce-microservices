package converter

import (
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"github.com/stretchr/testify/require"
)

func TestToEntityStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   notificationsv1.OrderStatus
		want entity.OrderStatus
	}{
		{name: "new", in: notificationsv1.OrderStatus_ORDER_STATUS_NEW, want: entity.OrderStatusNew},
		{name: "awaiting payment", in: notificationsv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT, want: entity.OrderStatusAwaitingPayment},
		{name: "failed", in: notificationsv1.OrderStatus_ORDER_STATUS_FAILED, want: entity.OrderStatusFailed},
		{name: "paid", in: notificationsv1.OrderStatus_ORDER_STATUS_PAID, want: entity.OrderStatusPaid},
		{name: "cancelled", in: notificationsv1.OrderStatus_ORDER_STATUS_CANCELLED, want: entity.OrderStatusCancelled},
		{name: "unspecified", in: notificationsv1.OrderStatus_ORDER_STATUS_UNSPECIFIED, want: entity.OrderStatusUnavailable},
		{name: "unknown", in: notificationsv1.OrderStatus(100), want: entity.OrderStatusUnavailable},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, ToEntityStatus(tt.in))
		})
	}
}

func TestToEntityMessage(t *testing.T) {
	t.Parallel()

	require.Equal(t, entity.Message{
		UserID:  42,
		OrderID: 777,
		Status:  entity.OrderStatusPaid,
	}, ToEntityMessage(42, 777, notificationsv1.OrderStatus_ORDER_STATUS_PAID))
}
