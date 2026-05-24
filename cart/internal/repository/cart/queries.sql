-- name: AddItemInCartChecked :execrows
INSERT INTO cart.users_carts (user_id, sku, amount)
VALUES (sqlc.arg(user_id), sqlc.arg(sku), sqlc.arg(amount)::bigint)
ON CONFLICT (user_id, sku) DO UPDATE
SET amount = cart.users_carts.amount + EXCLUDED.amount
WHERE cart.users_carts.amount + EXCLUDED.amount <= sqlc.arg(stock)::bigint;

-- name: DeleteItemFromCart :exec
DELETE
FROM cart.users_carts
WHERE user_id = $1
  AND sku = $2;

-- name: ClearUserCart :exec
DELETE
FROM cart.users_carts
WHERE user_id = $1;

-- name: GetUserCart :many
SELECT sku, amount
FROM cart.users_carts
WHERE user_id = $1;

-- name: GetUserCartForUpdate :many
SELECT sku, amount
FROM cart.users_carts
WHERE user_id = $1
FOR UPDATE;

-- name: GetItemCountInCart :one
SELECT COALESCE(
               (SELECT amount
                FROM cart.users_carts
                WHERE user_id = $1
                  AND sku = $2),
               0
       )::bigint AS amount;