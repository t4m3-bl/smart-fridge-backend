-- name: CreateProduct :exec
INSERT INTO products (
    id, user_id, name, category_id, description, quantity, image_link,
    manufactured_at, expires_at, opened_at, shelf_life_after_opening
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: UpdateProduct :execrows
UPDATE products
SET name                     = $3,
    category_id              = $4,
    description              = $5,
    quantity                 = $6,
    image_link               = $7,
    manufactured_at          = $8,
    expires_at               = $9,
    opened_at                = $10,
    shelf_life_after_opening = $11,
    updated_at               = now()
WHERE id = $1 AND user_id = $2;

-- name: DeleteProduct :execrows
DELETE FROM products
WHERE id = $1 AND user_id = $2;
