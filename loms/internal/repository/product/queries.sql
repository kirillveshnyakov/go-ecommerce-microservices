-- name: GetProductBySKU :one
SELECT name, price
FROM loms.products
WHERE sku = $1;

-- name: AddProduct :one
INSERT INTO loms.products (name, price)
VALUES ($1, $2) RETURNING sku;