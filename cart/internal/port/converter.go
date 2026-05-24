package port

import (
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	lomsv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
)

func FromPortToLomsItem(item Item) *lomsv1.Item {
	return &lomsv1.Item{
		Sku:   uint32(item.SKU),
		Count: item.Count,
	}
}

func FromPortToLomsItemsArray(items []Item) []*lomsv1.Item {
	lomsItems := make([]*lomsv1.Item, len(items))
	for i, item := range items {
		lomsItems[i] = FromPortToLomsItem(item)
	}
	return lomsItems
}

func FromEntityToPortItem(item entity.Item) Item {
	return Item{
		SKU:   item.SKU,
		Count: item.Count,
	}
}

func FromEntityToPortItemsArray(items []entity.Item) []Item {
	Items := make([]Item, len(items))
	for i, item := range items {
		Items[i] = FromEntityToPortItem(item)
	}
	return Items
}
