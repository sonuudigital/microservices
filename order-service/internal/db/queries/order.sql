-- name: CreateOrder :one
INSERT INTO orders (user_id, total_amount)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2
WHERE id = $1
RETURNING *;
