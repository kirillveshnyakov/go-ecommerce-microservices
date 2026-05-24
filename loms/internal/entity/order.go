package entity

import (
	"time"
)

type UserID int64
type OrderID int64
type OrderStatus string

const (
	OrderStatusUnavailable     OrderStatus = "unavailable"
	OrderStatusNew             OrderStatus = "new"
	OrderStatusAwaitingPayment OrderStatus = "awaiting_payment"
	OrderStatusFailed          OrderStatus = "failed"
	OrderStatusPaid            OrderStatus = "paid"
	OrderStatusCancelled       OrderStatus = "cancelled"
)

type Item struct {
	SKU   SKU
	Count uint64
}

type Order struct {
	ID        OrderID
	UserID    UserID
	Status    OrderStatus
	Items     []Item
	CreatedAt time.Time
	UpdatedAt time.Time
}
