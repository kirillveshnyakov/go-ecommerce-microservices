package converter

import (
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	lomssv "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
)

func ToOrderStatus(status lomssv.OrderStatus) entity.OrderStatus {
	switch status {
	case lomssv.OrderStatus_ORDER_STATUS_NEW:
		return entity.OrderStatusNew
	case lomssv.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT:
		return entity.OrderStatusAwaitingPayment
	case lomssv.OrderStatus_ORDER_STATUS_FAILED:
		return entity.OrderStatusFailed
	case lomssv.OrderStatus_ORDER_STATUS_PAID:
		return entity.OrderStatusPaid
	case lomssv.OrderStatus_ORDER_STATUS_CANCELLED:
		return entity.OrderStatusCancelled
	}
	return entity.OrderStatusUnavailable
}

func FromOrderStatus(orderStatus entity.OrderStatus) lomssv.OrderStatus {
	switch orderStatus {
	case entity.OrderStatusNew:
		return lomssv.OrderStatus_ORDER_STATUS_NEW
	case entity.OrderStatusAwaitingPayment:
		return lomssv.OrderStatus_ORDER_STATUS_AWAITING_PAYMENT
	case entity.OrderStatusFailed:
		return lomssv.OrderStatus_ORDER_STATUS_FAILED
	case entity.OrderStatusPaid:
		return lomssv.OrderStatus_ORDER_STATUS_PAID
	case entity.OrderStatusCancelled:
		return lomssv.OrderStatus_ORDER_STATUS_CANCELLED
	}
	return lomssv.OrderStatus_ORDER_STATUS_UNSPECIFIED
}

func ToEntityItem(item *lomssv.Item) entity.Item {
	return entity.Item{
		SKU:   entity.SKU(item.GetSku()),
		Count: uint64(item.GetCount()),
	}
}

func ToEntityItemsArray(items []*lomssv.Item) []entity.Item {
	entityItems := make([]entity.Item, len(items))
	for i, item := range items {
		entityItems[i] = ToEntityItem(item)
	}
	return entityItems
}

func FromEntityItem(item entity.Item) *lomssv.Item {
	return &lomssv.Item{
		Sku:   uint32(item.SKU),
		Count: uint32(item.Count),
	}
}

func FromEntityItemsArray(items []entity.Item) []*lomssv.Item {
	lomsItems := make([]*lomssv.Item, len(items))
	for i, item := range items {
		lomsItems[i] = FromEntityItem(item)
	}
	return lomsItems
}
