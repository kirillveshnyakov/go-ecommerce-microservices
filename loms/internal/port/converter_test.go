package port

import (
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
	"github.com/stretchr/testify/require"
)

func TestFromPortToProtoStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   OrderStatus
		want notificationsv1.OrderStatus
	}{
		{
			name: "new",
			in:   OrderStatusNew,
			want: notificationsv1.OrderStatus_ORDER_STATUS_NEW,
		},
		{
			name: "awaiting payment",
			in:   OrderStatusAwaitingPayment,
			want: notificationsv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT,
		},
		{
			name: "failed",
			in:   OrderStatusFailed,
			want: notificationsv1.OrderStatus_ORDER_STATUS_FAILED,
		},
		{
			name: "paid",
			in:   OrderStatusPaid,
			want: notificationsv1.OrderStatus_ORDER_STATUS_PAID,
		},
		{
			name: "cancelled",
			in:   OrderStatusCancelled,
			want: notificationsv1.OrderStatus_ORDER_STATUS_CANCELLED,
		},
		{
			name: "unknown",
			in:   OrderStatus("unknown"),
			want: notificationsv1.OrderStatus_ORDER_STATUS_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FromPortToProtoStatus(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFromEntityToPortStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   entity.OrderStatus
		want OrderStatus
	}{
		{
			name: "new",
			in:   entity.OrderStatusNew,
			want: OrderStatusNew,
		},
		{
			name: "awaiting payment",
			in:   entity.OrderStatusAwaitingPayment,
			want: OrderStatusAwaitingPayment,
		},
		{
			name: "failed",
			in:   entity.OrderStatusFailed,
			want: OrderStatusFailed,
		},
		{
			name: "paid",
			in:   entity.OrderStatusPaid,
			want: OrderStatusPaid,
		},
		{
			name: "cancelled",
			in:   entity.OrderStatusCancelled,
			want: OrderStatusCancelled,
		},
		{
			name: "unknown",
			in:   entity.OrderStatus("unknown"),
			want: OrderStatusUnspecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FromEntityToPortStatus(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}
