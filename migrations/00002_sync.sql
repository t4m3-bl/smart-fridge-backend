-- +goose Up
CREATE TABLE categories (
    id      uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name    text NOT NULL CHECK (btrim(name) <> ''),
    CONSTRAINT categories_user_name_uq UNIQUE (user_id, name),
    CONSTRAINT categories_id_user_uq   UNIQUE (id, user_id)
);

ALTER TABLE products DROP COLUMN category;
ALTER TABLE products ADD COLUMN category_id uuid;
ALTER TABLE products ADD CONSTRAINT products_category_fk
    FOREIGN KEY (category_id, user_id) REFERENCES categories (id, user_id) ON DELETE SET NULL (category_id);

CREATE TABLE mutations (
    seq        bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    id         uuid        NOT NULL,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type       text        NOT NULL CHECK (type IN ('createProduct', 'updateProduct', 'deleteProduct')),
    product_id uuid        NOT NULL,
    payload    jsonb,
    status     text        NOT NULL CHECK (status IN ('applied', 'rejected')),
    reason     text,
    message    text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT mutations_user_id_uq UNIQUE (user_id, id),
    CHECK ((status = 'applied') = (reason IS NULL))
);

CREATE INDEX mutations_user_applied_seq_idx ON mutations (user_id, seq) WHERE status = 'applied';

-- +goose Down
DROP TABLE mutations;

ALTER TABLE products DROP CONSTRAINT products_category_fk;
ALTER TABLE products DROP COLUMN category_id;
ALTER TABLE products ADD COLUMN category text;

DROP TABLE categories;
