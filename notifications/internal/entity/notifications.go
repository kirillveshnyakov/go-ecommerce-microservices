package entity

type OrderStatus string

const (
	OrderStatusUnavailable     OrderStatus = "unavailable"
	OrderStatusNew             OrderStatus = "new"
	OrderStatusAwaitingPayment OrderStatus = "awaiting_payment"
	OrderStatusFailed          OrderStatus = "failed"
	OrderStatusPaid            OrderStatus = "paid"
	OrderStatusCancelled       OrderStatus = "cancelled"
)

type Message struct {
	UserID  int64
	OrderID int64
	Status  OrderStatus
}
