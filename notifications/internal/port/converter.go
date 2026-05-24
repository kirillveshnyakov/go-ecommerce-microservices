package port

import (
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/entity"
)

func FromEntityToPortClientMessage(message entity.Message) CallbackPayload {
	return CallbackPayload{
		UserID:  message.UserID,
		OrderID: message.OrderID,
		Status:  FromEntityToPortStatus(message.Status),
	}
}

func FromPortToEntityClientMessage(message CallbackPayload) entity.Message {
	return entity.Message{
		UserID:  message.UserID,
		OrderID: message.OrderID,
		Status:  FromPortToEntityStatus(message.Status),
	}
}

func FromEntityToPortKafkaMessage(message entity.Message) KafkaMessage {
	return KafkaMessage{
		UserID:  message.UserID,
		OrderID: message.OrderID,
		Status:  FromEntityToPortStatus(message.Status),
	}
}

func FromPortToEntityKafkaMessage(message KafkaMessage) entity.Message {
	return entity.Message{
		UserID:  message.UserID,
		OrderID: message.OrderID,
		Status:  FromPortToEntityStatus(message.Status),
	}
}

func FromEntityToPortStatus(status entity.OrderStatus) string {
	switch status {
	case entity.OrderStatusNew:
		return StatusNew
	case entity.OrderStatusAwaitingPayment:
		return StatusAwaitingPayment
	case entity.OrderStatusFailed:
		return StatusFailed
	case entity.OrderStatusPaid:
		return StatusPaid
	case entity.OrderStatusCancelled:
		return StatusCancelled
	default:
		return StatusUnavailable
	}
}

func FromPortToEntityStatus(status string) entity.OrderStatus {
	switch status {
	case StatusNew:
		return entity.OrderStatusNew
	case StatusAwaitingPayment:
		return entity.OrderStatusAwaitingPayment
	case StatusFailed:
		return entity.OrderStatusFailed
	case StatusPaid:
		return entity.OrderStatusPaid
	case StatusCancelled:
		return entity.OrderStatusCancelled
	default:
		return entity.OrderStatusUnavailable
	}
}
