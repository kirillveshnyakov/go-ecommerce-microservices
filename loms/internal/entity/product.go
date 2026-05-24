package entity

type SKU uint32

type Product struct {
	ID    SKU
	Name  string
	Price uint32
}
