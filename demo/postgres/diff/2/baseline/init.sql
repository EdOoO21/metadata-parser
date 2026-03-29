DROP SCHEMA IF EXISTS warehouse CASCADE;

CREATE SCHEMA warehouse;

CREATE TABLE warehouse.products (
    sku TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    active BOOLEAN NOT NULL
);

CREATE TABLE warehouse.stock_balances (
    sku TEXT NOT NULL REFERENCES warehouse.products(sku),
    warehouse_code TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    updated_at DATE NOT NULL,
    PRIMARY KEY (sku, warehouse_code)
);

CREATE TABLE warehouse.deliveries (
    delivery_id BIGINT PRIMARY KEY,
    sku TEXT NOT NULL REFERENCES warehouse.products(sku),
    planned_date DATE NOT NULL,
    vendor TEXT NOT NULL
);

INSERT INTO warehouse.products (sku, name, active) VALUES
    ('SKU-100', 'Office Chair', TRUE),
    ('SKU-200', 'Desk Lamp', TRUE),
    ('SKU-300', 'Archive Box', FALSE);

INSERT INTO warehouse.stock_balances (sku, warehouse_code, quantity, updated_at) VALUES
    ('SKU-100', 'MOW', 12, '2026-03-10'),
    ('SKU-200', 'KZN', 5, '2026-03-10'),
    ('SKU-300', 'NNV', 0, '2026-03-10');

INSERT INTO warehouse.deliveries (delivery_id, sku, planned_date, vendor) VALUES
    (7001, 'SKU-100', '2026-03-15', 'north-supply'),
    (7002, 'SKU-200', '2026-03-16', 'light-market');
