-- name: GetProduct :one
SELECT * FROM products
WHERE user_id = @user_id AND id = @id;

-- name: ListProducts :many
SELECT * FROM products
WHERE user_id = @user_id
ORDER BY id;

-- name: CreateProduct :exec
INSERT INTO products (
    user_id, id, name, category_id, description, quantity, image_link,
    manufactured_at, expires_at, opened_at, shelf_life_after_opening
) VALUES (
    @user_id, @id, @name, @category_id, @description, @quantity, @image_link,
    @manufactured_at, @expires_at, @opened_at, @shelf_life_after_opening
);

-- name: UpdateProduct :execrows
UPDATE products SET
    name                     = @name,
    category_id              = @category_id,
    description              = @description,
    quantity                 = @quantity,
    image_link               = @image_link,
    manufactured_at          = @manufactured_at,
    expires_at               = @expires_at,
    opened_at                = @opened_at,
    shelf_life_after_opening = @shelf_life_after_opening,
    version                  = version + 1,
    updated_at               = now()
WHERE user_id = @user_id
  AND id = @id
  AND version = @base_version::int;

-- name: DeleteProduct :execrows
DELETE FROM products
WHERE user_id = @user_id
  AND id = @id
  AND version = @base_version::int;

