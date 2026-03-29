CREATE SCHEMA IF NOT EXISTS sales;

CREATE TABLE IF NOT EXISTS sales.orders (
    id BIGINT PRIMARY KEY,
    customer_email TEXT,
    amount NUMERIC(12,2) NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    paid_at TIMESTAMP
);

COMMENT ON TABLE sales.orders IS 'Sales orders fixture';
COMMENT ON COLUMN sales.orders.customer_email IS 'Customer email';

INSERT INTO sales.orders (id, customer_email, amount, status, created_at, paid_at)
VALUES
    (1, 'buyer1@example.com', 1250.50, 'paid', '2025-01-10 10:00:00', '2025-01-10 11:00:00'),
    (2, 'buyer2@example.com', 320.00, 'created', '2025-01-11 09:30:00', NULL),
    (3, NULL, 780.25, 'cancelled', '2025-01-12 16:45:00', NULL)
ON CONFLICT (id) DO NOTHING;

CREATE OR REPLACE VIEW sales.paid_orders AS
SELECT
    id,
    customer_email,
    amount,
    paid_at
FROM sales.orders
WHERE paid_at IS NOT NULL;
