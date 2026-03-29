DROP SCHEMA IF EXISTS sales CASCADE;

CREATE SCHEMA sales;

CREATE TABLE sales.customers (
    customer_id BIGINT PRIMARY KEY,
    full_name TEXT NOT NULL,
    segment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE sales.orders (
    order_id BIGINT PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES sales.customers(customer_id),
    total_amount NUMERIC(12, 2) NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE VIEW sales.customer_totals AS
SELECT
    c.customer_id,
    c.full_name,
    COUNT(o.order_id) AS orders_count,
    SUM(o.total_amount) AS total_amount
FROM sales.customers c
LEFT JOIN sales.orders o ON o.customer_id = c.customer_id
GROUP BY c.customer_id, c.full_name;

INSERT INTO sales.customers (customer_id, full_name, segment, created_at) VALUES
    (1, 'Ivan Petrov', 'vip', '2026-03-01T10:00:00Z'),
    (2, 'Anna Sidorova', 'base', '2026-03-02T11:00:00Z'),
    (3, 'Pavel Smirnov', 'vip', '2026-03-03T12:00:00Z');

INSERT INTO sales.orders (order_id, customer_id, total_amount, status, created_at) VALUES
    (101, 1, 1250.50, 'paid', '2026-03-10T10:00:00Z'),
    (102, 2, 320.00, 'created', '2026-03-11T11:00:00Z'),
    (103, 1, 780.25, 'cancelled', '2026-03-12T12:00:00Z');
