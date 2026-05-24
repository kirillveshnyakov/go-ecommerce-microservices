-- +goose Up
-- +goose StatementBegin
CREATE TABLE loms.available_stocks
(
    sku    BIGINT NOT NULL PRIMARY KEY REFERENCES loms.products (sku) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount >= 0)
);

CREATE TABLE loms.reserved_stocks
(
    sku      BIGINT NOT NULL REFERENCES loms.products (sku) ON DELETE CASCADE,
    order_id BIGINT NOT NULL REFERENCES loms.orders (id) ON DELETE CASCADE,
    amount   BIGINT NOT NULL CHECK (amount >= 0),

    PRIMARY KEY (sku, order_id)
);

CREATE INDEX idx_reserved_stocks_order_id ON loms.reserved_stocks USING BTREE (order_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS loms.idx_reserved_stocks_order_id;
DROP TABLE IF EXISTS loms.reserved_stocks;
DROP TABLE IF EXISTS loms.available_stocks;
-- +goose StatementEnd
