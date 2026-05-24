-- name: GetAvailableStock :one
SELECT COALESCE(loms.available_stocks.amount, 0) ::bigint AS amount
FROM loms.available_stocks
WHERE loms.available_stocks.sku = $1;

-- name: SetAvailableStock :execrows
INSERT INTO loms.available_stocks (sku, amount)
SELECT loms.products.sku, $2
FROM loms.products
WHERE loms.products.sku = $1 ON CONFLICT (sku) DO
UPDATE SET amount = EXCLUDED.amount;

-- name: AddAvailableStock :exec
INSERT INTO loms.available_stocks (sku, amount)
VALUES ($1, $2) ON CONFLICT (sku) DO
UPDATE SET amount = loms.available_stocks.amount + EXCLUDED.amount;

-- name: AddReserveStock :exec
INSERT INTO loms.reserved_stocks (sku, order_id, amount)
VALUES ($1, $2, $3) ON CONFLICT (sku, order_id) DO
UPDATE SET amount = loms.reserved_stocks.amount + EXCLUDED.amount;

-- name: DecrementAvailableStock :execrows
UPDATE loms.available_stocks
SET amount = amount - $2
WHERE sku = $1
  AND amount >= $2;

-- name: DecrementReservedStock :execrows
UPDATE loms.reserved_stocks
SET amount = amount - $3
WHERE sku = $1
  AND order_id = $2
  and amount >= $3;