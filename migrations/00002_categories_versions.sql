-- +goose Up
CREATE TABLE categories (
    id      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name    text NOT NULL CHECK (btrim(name) <> ''),
    version integer NOT NULL DEFAULT 1 CHECK (version > 0),
    CONSTRAINT categories_user_name_uq UNIQUE (user_id, name),
    CONSTRAINT categories_id_user_uq UNIQUE (id, user_id)
);

INSERT INTO categories (user_id, name)
SELECT DISTINCT user_id, btrim(category)
FROM products
WHERE category IS NOT NULL AND btrim(category) <> '';

ALTER TABLE products ADD COLUMN category_id uuid;

UPDATE products p
SET category_id = c.id
FROM categories c
WHERE c.user_id = p.user_id AND c.name = btrim(p.category);

ALTER TABLE products DROP COLUMN category;
ALTER TABLE products ADD CONSTRAINT products_category_fk
    FOREIGN KEY (category_id, user_id) REFERENCES categories (id, user_id)
    ON DELETE SET NULL (category_id);
CREATE INDEX products_user_category_idx ON products (user_id, category_id);

ALTER TABLE products ADD COLUMN version integer NOT NULL DEFAULT 1 CHECK (version > 0);
ALTER TABLE settings ADD COLUMN version integer NOT NULL DEFAULT 1 CHECK (version > 0);

-- +goose Down
ALTER TABLE settings DROP COLUMN version;
ALTER TABLE products DROP COLUMN version;
ALTER TABLE products ADD COLUMN category text;

UPDATE products p
SET category = c.name
FROM categories c
WHERE c.id = p.category_id AND c.user_id = p.user_id;

ALTER TABLE products DROP CONSTRAINT products_category_fk;
DROP INDEX products_user_category_idx;
ALTER TABLE products DROP COLUMN category_id;
DROP TABLE categories;
