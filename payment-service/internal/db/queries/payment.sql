-- name: GetPaymentByID :one
SELECT * FROM payments
WHERE id = $1;

-- name: CreatePayment :one
INSERT INTO payments (order_id, user_id, amount, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdatePaymentStatus :exec
UPDATE payments
SET status = $2
WHERE id = $1;
