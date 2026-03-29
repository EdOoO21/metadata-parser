DROP SCHEMA IF EXISTS warehouse CASCADE;

CREATE SCHEMA warehouse;

CREATE TABLE warehouse.products (
    sku TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    active TEXT NOT NULL,
    price NUMERIC(12, 2),
    deactivation_reason TEXT
);

CREATE TABLE warehouse.stock_balances (
    sku TEXT NOT NULL REFERENCES warehouse.products(sku),
    warehouse_code TEXT NOT NULL,
    quantity INTEGER,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (sku, warehouse_code)
);

CREATE TABLE warehouse.replenishments (
    replenishment_id BIGINT PRIMARY KEY,
    sku TEXT NOT NULL REFERENCES warehouse.products(sku),
    expected_at TIMESTAMPTZ NOT NULL,
    quantity INTEGER NOT NULL
);

INSERT INTO warehouse.products (sku, name, active, price, deactivation_reason) VALUES
    ('SKU-100', 'Office Chair', 'active', 12990.00, NULL),
    ('SKU-200', 'Desk Lamp', 'active', 3490.00, NULL),
    ('SKU-300', 'Archive Box', 'archived', 490.00, 'seasonal');

INSERT INTO warehouse.stock_balances (sku, warehouse_code, quantity, updated_at) VALUES
    ('SKU-100', 'MOW', 12, '2026-03-10T09:00:00Z'),
    ('SKU-200', 'KZN', NULL, '2026-03-10T09:00:00Z'),
    ('SKU-300', 'NNV', 1, '2026-03-10T09:00:00Z');

INSERT INTO warehouse.replenishments (replenishment_id, sku, expected_at, quantity) VALUES
    (8101, 'SKU-100', '2026-03-20T07:00:00Z', 20),
    (8102, 'SKU-300', '2026-03-21T09:30:00Z', 15);
