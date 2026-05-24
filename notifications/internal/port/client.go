package port

var (
	StatusNew             = "new"
	StatusAwaitingPayment = "awaiting_payment"
	StatusFailed          = "failed"
	StatusPaid            = "paid"
	StatusCancelled       = "cancelled"
	StatusUnavailable     = "unavailable"
)

type CallbackPayload struct {
	UserID  int64  `json:"user_id"`
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
}
