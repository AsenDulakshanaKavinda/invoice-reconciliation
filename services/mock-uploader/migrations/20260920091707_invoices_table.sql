-- +goose Up
CREATE TABLE invoices (
    id                UUID PRIMARY KEY,
    object_key        TEXT        NOT NULL UNIQUE,
    original_filename TEXT        NOT NULL,
    content_type      TEXT        NOT NULL,
    declared_size     BIGINT      NOT NULL,
    size_bytes        BIGINT,
    status            TEXT        NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'uploaded', 'failed')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    uploaded_at       TIMESTAMPTZ
);


-- +goose Down
DROP TABLE invoices;
