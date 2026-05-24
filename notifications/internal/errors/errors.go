package errors

import "errors"

var (
	ErrSendNotification  = errors.New("send notification failed")
	ErrEmptyCallbackAddr = errors.New("callback address is empty")
)
