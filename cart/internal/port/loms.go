package port

import "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/entity"

type Item struct {
	SKU   entity.SKU
	Count uint32
}

type CreateOrderRequest struct {
	UserID entity.UserID
	Items  []Item
}
