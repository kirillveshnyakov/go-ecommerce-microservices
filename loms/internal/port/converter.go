package port

import (
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
)

func FromPortToProtoStatus(status OrderStatus) notificationsv1.OrderStatus {
	switch status {
	case OrderStatusNew:
		return notificationsv1.OrderStatus_ORDER_STATUS_NEW
	case OrderStatusAwaitingPayment:
		return notificationsv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT
	case OrderStatusFailed:
		return notificationsv1.OrderStatus_ORDER_STATUS_FAILED
	case OrderStatusPaid:
		return notificationsv1.OrderStatus_ORDER_STATUS_PAID
	case OrderStatusCancelled:
		return notificationsv1.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return notificationsv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func FromEntityToPortStatus(status entity.OrderStatus) OrderStatus {
	switch status {
	case entity.OrderStatusNew:
		return OrderStatusNew
	case entity.OrderStatusAwaitingPayment:
		return OrderStatusAwaitingPayment
	case entity.OrderStatusFailed:
		return OrderStatusFailed
	case entity.OrderStatusPaid:
		return OrderStatusPaid
	case entity.OrderStatusCancelled:
		return OrderStatusCancelled
	default:
		return OrderStatusUnspecified
	}
}
