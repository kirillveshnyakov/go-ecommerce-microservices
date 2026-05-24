package order

import (
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/order/sqlc"
	"github.com/stretchr/testify/require"
)

func TestToEntityStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   sqlc.LomsOrderStatus
		want entity.OrderStatus
	}{
		{
			name: "new",
			in:   sqlc.LomsOrderStatusNew,
			want: entity.OrderStatusNew,
		},
		{
			name: "awaiting payment",
			in:   sqlc.LomsOrderStatusAwaitingPayment,
			want: entity.OrderStatusAwaitingPayment,
		},
		{
			name: "failed",
			in:   sqlc.LomsOrderStatusFailed,
			want: entity.OrderStatusFailed,
		},
		{
			name: "paid",
			in:   sqlc.LomsOrderStatusPaid,
			want: entity.OrderStatusPaid,
		},
		{
			name: "cancelled",
			in:   sqlc.LomsOrderStatusCancelled,
			want: entity.OrderStatusCancelled,
		},
		{
			name: "unknown",
			in:   sqlc.LomsOrderStatus("unknown"),
			want: entity.OrderStatusUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ToEntityStatus(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFromEntityStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   entity.OrderStatus
		want sqlc.LomsOrderStatus
	}{
		{
			name: "new",
			in:   entity.OrderStatusNew,
			want: sqlc.LomsOrderStatusNew,
		},
		{
			name: "awaiting payment",
			in:   entity.OrderStatusAwaitingPayment,
			want: sqlc.LomsOrderStatusAwaitingPayment,
		},
		{
			name: "failed",
			in:   entity.OrderStatusFailed,
			want: sqlc.LomsOrderStatusFailed,
		},
		{
			name: "paid",
			in:   entity.OrderStatusPaid,
			want: sqlc.LomsOrderStatusPaid,
		},
		{
			name: "cancelled",
			in:   entity.OrderStatusCancelled,
			want: sqlc.LomsOrderStatusCancelled,
		},
		{
			name: "unknown",
			in:   entity.OrderStatus("unknown"),
			want: sqlc.LomsOrderStatusUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FromEntityStatus(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}
