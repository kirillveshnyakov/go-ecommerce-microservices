package errors

import (
	"errors"
)

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidStatus     = errors.New("invalid order status for operation")
	ErrSendNotification  = errors.New("send notification failed")
)
