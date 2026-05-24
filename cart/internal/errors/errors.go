package errors

import "errors"

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrUserCartNotFound  = errors.New("user cart not found or empty")
)
