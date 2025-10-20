-- name: GetProduct :one
SELECT * FROM products
WHERE id = $1;

-- name: ListProductsPaginated :many
SELECT * FROM products
ORDER BY name
LIMIT $1
OFFSET $2;

-- name: CreateProduct :one
INSERT INTO products (
  name,
  description,
  price,
  code,
  stock_quantity
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET
  name = $2,
  description = $3,
  price = $4,
  code = $5,
  stock_quantity = $6,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;

-- name: GetProductsByIDs :many
SELECT * FROM products
WHERE id = ANY(@product_ids::uuid[]);
