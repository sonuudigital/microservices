-- name: CreateOrder :one
INSERT INTO orders (user_id, total_amount)
VALUES ($1, $2)
RETURNING *;

-- name: GetOrderById :one
SELECT id, user_id, total_amount, status
FROM orders
WHERE id = $1;

-- name: UpdateOrderStatus :one
UPDATE orders
SET status = $2
WHERE id = $1
RETURNING *;

-- name: GetOrderStatusByName :one
SELECT id, name
FROM order_statuses
WHERE name = $1;
