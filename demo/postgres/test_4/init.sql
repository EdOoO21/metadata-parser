CREATE SCHEMA IF NOT EXISTS warehouse;

CREATE TABLE IF NOT EXISTS warehouse.products (
    id BIGINT PRIMARY KEY,
    sku TEXT NOT NULL,
    product_name TEXT NOT NULL,
    category TEXT,
    price NUMERIC(10,2) NOT NULL,
    active BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS warehouse.stock_movements (
    id BIGINT PRIMARY KEY,
    product_id BIGINT NOT NULL,
    direction TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    moved_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_stock_product
        FOREIGN KEY (product_id) REFERENCES warehouse.products (id)
);

INSERT INTO warehouse.products (id, sku, product_name, category, price, active)
VALUES
    (1, 'SKU-100', 'Office Chair', 'furniture', 12990.00, TRUE),
    (2, 'SKU-101', 'Desk Lamp', 'lighting', 3490.00, TRUE),
    (3, 'SKU-102', 'Archive Box', NULL, 490.00, FALSE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO warehouse.stock_movements (id, product_id, direction, quantity, moved_at)
VALUES
    (1, 1, 'in', 20, '2025-02-01 09:00:00'),
    (2, 1, 'out', 3, '2025-02-03 12:00:00'),
    (3, 2, 'in', 50, '2025-02-02 10:30:00'),
    (4, 3, 'out', 5, '2025-02-04 15:10:00')
ON CONFLICT (id) DO NOTHING;
