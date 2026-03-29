CREATE SCHEMA IF NOT EXISTS accounting;

CREATE TABLE IF NOT EXISTS accounting.customers (
    id BIGINT PRIMARY KEY,
    full_name TEXT NOT NULL,
    email TEXT,
    city TEXT,
    registered_at TIMESTAMP
);

INSERT INTO accounting.customers (id, full_name, email, city, registered_at)
VALUES
    (1, 'Ivan Petrov', 'ivan.petrov@example.com', 'Moscow', '2024-01-10 09:00:00'),
    (2, 'Anna Sidorova', 'anna.sidorova@example.com', 'Kazan', '2024-01-15 11:30:00'),
    (3, 'Pavel Smirnov', NULL, 'Tula', '2024-02-01 14:45:00')
ON CONFLICT (id) DO NOTHING;

CREATE VIEW accounting.customer_contacts AS
SELECT
    id,
    full_name,
    email
FROM accounting.customers;
