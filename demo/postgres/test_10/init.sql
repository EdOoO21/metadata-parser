CREATE SCHEMA IF NOT EXISTS marketing;

CREATE TABLE IF NOT EXISTS marketing.leads (
    id BIGINT PRIMARY KEY,
    contact_name TEXT,
    email TEXT,
    phone TEXT,
    source_channel TEXT,
    budget NUMERIC(12,2),
    contacted_at TIMESTAMP,
    converted BOOLEAN
);

INSERT INTO marketing.leads (id, contact_name, email, phone, source_channel, budget, contacted_at, converted)
VALUES
    (1, 'Kirill Andreev', 'kirill@example.com', NULL, 'ads', 15000.00, '2025-03-01 12:00:00', TRUE),
    (2, NULL, NULL, '+79995554433', 'seo', NULL, NULL, FALSE),
    (3, 'Elena Morozova', 'elena@example.com', '+79995550000', NULL, 5000.00, '2025-03-04 17:20:00', NULL)
ON CONFLICT (id) DO NOTHING;
