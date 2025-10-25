-- name: GetProductCategories :many
SELECT * FROM product_categories
ORDER BY name;

-- name: CreateProductCategory :one
INSERT INTO product_categories (
  name,
  description
) VALUES (
  $1, $2
)
RETURNING *;

-- name: UpdateProductCategory :exec
UPDATE product_categories
SET
  name = $2,
  description = $3,
  updated_at = NOW()
WHERE id = $1;

-- name: DeleteProductCategory :exec
DELETE FROM product_categories
WHERE id = $1;
