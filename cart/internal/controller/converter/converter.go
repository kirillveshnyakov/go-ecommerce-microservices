package converter

import (
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"
	cartv1 "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
)

func convertItemFromEntityToCart(item entity.ItemInfo) *cartv1.Item {
	return &cartv1.Item{
		Sku:   uint32(item.Item.SKU),
		Count: item.Item.Count,
		Name:  item.Name,
		Price: item.Price,
	}
}

func FromEntityToCartItems(items []entity.ItemInfo) []*cartv1.Item {
	lomsItems := make([]*cartv1.Item, len(items))
	for i, item := range items {
		lomsItems[i] = convertItemFromEntityToCart(item)
	}
	return lomsItems
}
