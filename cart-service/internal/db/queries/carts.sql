-- name: GetCartByUserID :one
SELECT * FROM carts
WHERE user_id = $1;

-- name: CreateCart :one
INSERT INTO carts (user_id)
VALUES ($1)
RETURNING *;

-- name: GetCartProductsByCartID :many
SELECT product_id, quantity
FROM carts_products
WHERE cart_id = $1;
