package converter

import (
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
	notificationsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/notifications/api/v1"
)

func ToEntityStatus(status notificationsv1.OrderStatus) entity.OrderStatus {
	switch status {
	case notificationsv1.OrderStatus_ORDER_STATUS_NEW:
		return entity.OrderStatusNew
	case notificationsv1.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT:
		return entity.OrderStatusAwaitingPayment
	case notificationsv1.OrderStatus_ORDER_STATUS_FAILED:
		return entity.OrderStatusFailed
	case notificationsv1.OrderStatus_ORDER_STATUS_PAID:
		return entity.OrderStatusPaid
	case notificationsv1.OrderStatus_ORDER_STATUS_CANCELLED:
		return entity.OrderStatusCancelled
	default:
		return entity.OrderStatusUnavailable
	}
}

func ToEntityMessage(userID int64, orderID int64, status notificationsv1.OrderStatus) entity.Message {
	return entity.Message{
		UserID:  userID,
		OrderID: orderID,
		Status:  ToEntityStatus(status),
	}
}
