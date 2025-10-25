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
  stock_quantity
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET
  name = $2,
  description = $3,
  price = $4,
  stock_quantity = $5,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProduct :exec
DELETE FROM products
WHERE id = $1;

-- name: GetProductsByIDs :many
SELECT * FROM products
WHERE id = ANY(@product_ids::uuid[]);

-- name: GetProductsByCategoryID :many
SELECT * FROM products
WHERE category_id = $1
ORDER BY id;
