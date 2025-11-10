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
  category_id,
  name,
  description,
  price,
  stock_quantity
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET
  category_id = $2,
  name = $3,
  description = $4,
  price = $5,
  stock_quantity = $6,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateStockBatch :execrows
UPDATE products
SET
  stock_quantity = products.stock_quantity - p.quantity,
  updated_at = NOW()
FROM
  json_to_recordset(sqlc.arg(update_params)::json) AS p("productId" uuid, quantity int)
WHERE
  products.id = p."productId";

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
