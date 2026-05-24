-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS cart;

CREATE TABLE cart.users_carts
(
    user_id BIGINT NOT NULL,
    sku     BIGINT NOT NULL,
    amount  BIGINT NOT NULL CHECK (amount > 0),

    PRIMARY KEY (user_id, sku)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cart.users_carts;
-- +goose StatementEnd
