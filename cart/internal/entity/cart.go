package entity

type UserID int64
type SKU uint32

type Item struct {
	SKU   SKU
	Count uint32
}

type ItemInfo struct {
	Item  Item
	Name  string
	Price uint32
}
