-- name: LockFridgeOwner :exec
SELECT id FROM users WHERE id = $1 FOR UPDATE;

-- name: FindProductDocument :one
SELECT to_jsonb(t) FROM products t WHERE id=$1 AND user_id=$2;

-- name: ListProductDocuments :many
SELECT to_jsonb(t) FROM products t WHERE user_id=$1 ORDER BY id LIMIT $2 OFFSET $3;

-- name: DeleteProductDocument :exec
DELETE FROM products WHERE id=$1 AND user_id=$2;

-- name: FindCategoryDocument :one
SELECT to_jsonb(t) FROM categories t WHERE id=$1 AND user_id=$2;

-- name: ListCategoryDocuments :many
SELECT to_jsonb(t) FROM categories t WHERE user_id=$1 ORDER BY id LIMIT $2 OFFSET $3;

-- name: DeleteCategoryDocument :exec
DELETE FROM categories WHERE id=$1 AND user_id=$2;

-- name: SaveProductDocument :one
INSERT INTO products (id,user_id,name,category_id,description,quantity,image_link,manufactured_at,expires_at,opened_at,shelf_life_after_opening,version)
SELECT id,user_id,name,category_id,description,quantity,image_link,manufactured_at,expires_at,opened_at,shelf_life_after_opening,version FROM jsonb_populate_record(NULL::products, sqlc.arg(document)::jsonb)
ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,category_id=EXCLUDED.category_id,description=EXCLUDED.description,quantity=EXCLUDED.quantity,image_link=EXCLUDED.image_link,manufactured_at=EXCLUDED.manufactured_at,expires_at=EXCLUDED.expires_at,opened_at=EXCLUDED.opened_at,shelf_life_after_opening=EXCLUDED.shelf_life_after_opening,version=EXCLUDED.version,updated_at=now() WHERE products.user_id=EXCLUDED.user_id
RETURNING to_jsonb(products.*);

-- name: SaveCategoryDocument :one
INSERT INTO categories (id,user_id,name,version)
SELECT id,user_id,name,version FROM jsonb_populate_record(NULL::categories, sqlc.arg(document)::jsonb)
ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,version=EXCLUDED.version WHERE categories.user_id=EXCLUDED.user_id
RETURNING to_jsonb(categories.*);

-- name: EnsureSettings :exec
INSERT INTO settings(user_id) VALUES($1) ON CONFLICT DO NOTHING;

-- name: GetSettingsDocument :one
SELECT to_jsonb(s) || jsonb_build_object('indicators', COALESCE((SELECT jsonb_agg(to_jsonb(i) ORDER BY color) FROM indicators i WHERE i.user_id=s.user_id),'[]'::jsonb)) FROM settings s WHERE s.user_id=$1;

-- name: SaveSettingsDocument :one
UPDATE settings SET has_notifications=d.has_notifications,notification_time=d.notification_time,timezone=d.timezone,version=d.version
FROM jsonb_populate_record(NULL::settings,sqlc.arg(document)::jsonb) d WHERE settings.user_id=d.user_id RETURNING to_jsonb(settings.*);

-- name: ClearIndicators :exec
DELETE FROM indicators WHERE user_id=$1;

-- name: AddIndicator :exec
INSERT INTO indicators(id,user_id,color,max_days) VALUES($1,$2,$3,$4);

-- name: ClearCategoryReferences :exec
UPDATE products SET category_id=NULL,version=version+1,updated_at=now() WHERE user_id=$1 AND category_id=$2;

