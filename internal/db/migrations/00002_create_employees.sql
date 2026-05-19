-- +goose Up
CREATE TABLE employees (
    id            BIGSERIAL PRIMARY KEY,
    department_id BIGINT NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    fullname      VARCHAR(255) NOT NULL,
    position      VARCHAR(255) NOT NULL,
    hired_at      DATE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX employees_department_id_idx ON employees (department_id);

-- +goose Down
DROP TABLE employees;
