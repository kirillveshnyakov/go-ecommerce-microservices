package converter

import (
	"testing"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomssv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	"github.com/stretchr/testify/require"
)

func TestToOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   lomssv.OrderStatus
		want entity.OrderStatus
	}{
		{
			name: "new",
			in:   lomssv.OrderStatus_ORDER_STATUS_NEW,
			want: entity.OrderStatusNew,
		},
		{
			name: "awaiting payment",
			in:   lomssv.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT,
			want: entity.OrderStatusAwaitingPayment,
		},
		{
			name: "failed",
			in:   lomssv.OrderStatus_ORDER_STATUS_FAILED,
			want: entity.OrderStatusFailed,
		},
		{
			name: "paid",
			in:   lomssv.OrderStatus_ORDER_STATUS_PAID,
			want: entity.OrderStatusPaid,
		},
		{
			name: "cancelled",
			in:   lomssv.OrderStatus_ORDER_STATUS_CANCELLED,
			want: entity.OrderStatusCancelled,
		},
		{
			name: "unspecified",
			in:   lomssv.OrderStatus_ORDER_STATUS_UNSPECIFIED,
			want: entity.OrderStatusUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ToOrderStatus(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFromOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   entity.OrderStatus
		want lomssv.OrderStatus
	}{
		{
			name: "new",
			in:   entity.OrderStatusNew,
			want: lomssv.OrderStatus_ORDER_STATUS_NEW,
		},
		{
			name: "awaiting payment",
			in:   entity.OrderStatusAwaitingPayment,
			want: lomssv.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT,
		},
		{
			name: "failed",
			in:   entity.OrderStatusFailed,
			want: lomssv.OrderStatus_ORDER_STATUS_FAILED,
		},
		{
			name: "paid",
			in:   entity.OrderStatusPaid,
			want: lomssv.OrderStatus_ORDER_STATUS_PAID,
		},
		{
			name: "cancelled",
			in:   entity.OrderStatusCancelled,
			want: lomssv.OrderStatus_ORDER_STATUS_CANCELLED,
		},
		{
			name: "unavailable",
			in:   entity.OrderStatusUnavailable,
			want: lomssv.OrderStatus_ORDER_STATUS_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FromOrderStatus(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestItemsConversion(t *testing.T) {
	t.Parallel()

	t.Run("to entity items", func(t *testing.T) {
		t.Parallel()

		items := []*lomssv.Item{
			{Sku: 1001, Count: 2},
			{Sku: 1002, Count: 3},
		}
		want := []entity.Item{
			{SKU: 1001, Count: 2},
			{SKU: 1002, Count: 3},
		}

		got := ToEntityItemsArray(items)
		require.Equal(t, want, got)
	})

	t.Run("from entity items", func(t *testing.T) {
		t.Parallel()

		items := []entity.Item{
			{SKU: 2001, Count: 4},
			{SKU: 2002, Count: 5},
		}
		want := []*lomssv.Item{
			{Sku: 2001, Count: 4},
			{Sku: 2002, Count: 5},
		}

		got := FromEntityItemsArray(items)
		require.Equal(t, want, got)
	})
}
