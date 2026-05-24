-- name: AddOrder :one
INSERT INTO loms.orders (user_id, status)
VALUES ($1, $2)
RETURNING id;

-- name: GetOrder :one
SELECT id, user_id, status, created_at, updated_at
FROM loms.orders
WHERE id = $1;

-- name: GetOrderForUpdate :one
SELECT id, user_id, status, created_at, updated_at
FROM loms.orders
WHERE id = $1
FOR UPDATE;

-- name: AddOrderInfo :exec
INSERT INTO loms.order_info (order_id, sku, amount)
VALUES ($1, $2, $3)
ON CONFLICT (order_id, sku) DO UPDATE
    SET amount = loms.order_info.amount + EXCLUDED.amount;

-- name: GetOrderItems :many
SELECT sku, amount
FROM loms.order_info
WHERE order_id = $1;

-- name: GetOrderStatus :one
SELECT status
FROM loms.orders
WHERE id = $1;

-- name: SwapOrderStatus :execrows
UPDATE loms.orders
SET status = $3
WHERE id = $1
  AND status = $2;
