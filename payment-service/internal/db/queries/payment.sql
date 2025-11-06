-- name: GetPaymentByID :one
SELECT * FROM payments
WHERE id = $1;

-- name: CreatePayment :one
INSERT INTO payments (order_id, user_id, amount)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdatePaymentStatus :one
UPDATE payments
SET status = $2
WHERE id = $1
RETURNING *;

-- name: GetPaymentStatusByName :one
SELECT id FROM payment_statuses
WHERE name = $1;
