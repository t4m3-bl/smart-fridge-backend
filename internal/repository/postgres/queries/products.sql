-- name: GetProduct :one
SELECT * FROM products
WHERE user_id = @user_id AND id = @id;

-- name: ListProducts :many
SELECT * FROM products
WHERE user_id = @user_id
ORDER BY id;

-- name: CreateProduct :exec
INSERT INTO products (
    user_id, id, name, category_id, description, quantity,
    manufactured_at, expires_at, opened_at, shelf_life_after_opening
) VALUES (
    @user_id, @id, @name, @category_id, @description, @quantity,
    @manufactured_at, @expires_at, @opened_at, @shelf_life_after_opening
);

-- name: UpdateProduct :one
UPDATE products SET
    name                     = @name,
    category_id              = @category_id,
    description              = @description,
    quantity                 = @quantity,
    manufactured_at          = @manufactured_at,
    expires_at               = @expires_at,
    opened_at                = @opened_at,
    shelf_life_after_opening = @shelf_life_after_opening,
    version                  = version + 1,
    updated_at               = now()
WHERE user_id = @user_id
  AND id = @id
  AND version = @base_version::int
RETURNING image_key;

-- name: DeleteProduct :one
DELETE FROM products
WHERE user_id = @user_id
  AND id = @id
  AND version = @base_version::int
RETURNING image_key;

-- name: SetProductImage :one
UPDATE products
SET image_key = sqlc.narg(image_key), version = version + 1, updated_at = now()
WHERE user_id = @user_id AND id = @id AND version = @base_version::int
RETURNING version;
