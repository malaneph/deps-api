-- +goose Up
CREATE TABLE departments (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    parent_id  BIGINT REFERENCES departments(id) ON DELETE CASCADE,
    depth      INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique name among siblings; COALESCE maps NULL parent to 0 (no valid auto-increment ID)
CREATE UNIQUE INDEX departments_sibling_name_uidx
    ON departments (COALESCE(parent_id, 0), lower(name));

-- +goose Down
DROP TABLE departments;
