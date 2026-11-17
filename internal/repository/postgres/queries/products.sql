-- name: GetProduct :one
SELECT * FROM products
WHERE user_id = @user_id AND id = @id;

-- name: ListProducts :many
SELECT * FROM products
WHERE user_id = @user_id
ORDER BY id;

-- name: CreateProduct :one
INSERT INTO products (
    user_id, id, name, category_id, description, quantity, image_key,
    manufactured_at, expires_at, opened_at, shelf_life_after_opening
) VALUES (
    @user_id, @id, @name, @category_id, @description, @quantity, @image_key,
    @manufactured_at, @expires_at, @opened_at, @shelf_life_after_opening
)
ON CONFLICT (user_id, id) DO NOTHING
RETURNING *;

-- name: UpdateProduct :one
UPDATE products SET
    name                     = @name,
    category_id              = @category_id,
    description              = @description,
    quantity                 = @quantity,
    image_key                = @image_key,
    manufactured_at          = @manufactured_at,
    expires_at               = @expires_at,
    opened_at                = @opened_at,
    shelf_life_after_opening = @shelf_life_after_opening,
    version                  = version + 1,
    updated_at               = now()
WHERE user_id = @user_id
  AND id = @id
  AND version = @base_version::int
RETURNING *;

-- name: DeleteProduct :one
DELETE FROM products
WHERE user_id = @user_id
  AND id = @id
  AND version = @base_version::int
RETURNING *;

-- name: GetImageKey :one
SELECT image_key FROM products
WHERE user_id = @user_id
  AND id = @id;
