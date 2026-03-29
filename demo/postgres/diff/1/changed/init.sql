DROP SCHEMA IF EXISTS sales CASCADE;

CREATE SCHEMA sales;

CREATE TABLE sales.customers (
    customer_id BIGINT PRIMARY KEY,
    full_name TEXT NOT NULL,
    email TEXT,
    segment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

COMMENT ON COLUMN sales.customers.email IS 'Customer email';

CREATE TABLE sales.orders (
    order_id BIGINT PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES sales.customers(customer_id),
    total_amount TEXT NOT NULL,
    status TEXT NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE sales.returns (
    return_id BIGINT PRIMARY KEY,
    order_id BIGINT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

INSERT INTO sales.customers (customer_id, full_name, email, segment, created_at) VALUES
    (1, 'Ivan Petrov', 'ivan@example.com', 'vip', '2026-03-01T10:00:00Z'),
    (2, 'Anna Sidorova', NULL, 'base', '2026-03-02T11:00:00Z'),
    (3, 'Pavel Smirnov', 'pavel@example.com', 'vip', '2026-03-03T12:00:00Z');

INSERT INTO sales.orders (order_id, customer_id, total_amount, status, paid_at, created_at) VALUES
    (101, 1, '1250.50', 'paid', '2026-03-10T10:05:00Z', '2026-03-10T10:00:00Z'),
    (102, 2, '320.00', 'created', NULL, '2026-03-11T11:00:00Z'),
    (103, 1, '780.25', 'cancelled', NULL, '2026-03-12T12:00:00Z');

INSERT INTO sales.returns (return_id, order_id, reason, created_at) VALUES
    (901, 103, 'late_delivery', '2026-03-13T09:30:00Z');
