-- name: ListCategories :many
SELECT * FROM categories WHERE user_id = @user_id ORDER BY id;

-- name: CreateCategory :exec
INSERT INTO categories (user_id, id, name) VALUES (@user_id, @id, @name);

-- name: UpdateCategory :execrows
UPDATE categories SET name = @name, version = version + 1
WHERE user_id = @user_id AND id = @id AND version = @base_version::int;

-- name: DeleteCategory :execrows
DELETE FROM categories
WHERE user_id = @user_id AND id = @id AND version = @base_version::int;