CREATE SCHEMA IF NOT EXISTS billing;

CREATE TABLE IF NOT EXISTS billing.invoices (
    id BIGINT PRIMARY KEY,
    customer_name TEXT NOT NULL,
    total_amount NUMERIC(12,2) NOT NULL,
    issued_at DATE NOT NULL,
    due_at DATE NOT NULL,
    paid BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS billing.payments (
    id BIGINT PRIMARY KEY,
    invoice_id BIGINT NOT NULL,
    paid_amount NUMERIC(12,2) NOT NULL,
    payment_method TEXT NOT NULL,
    paid_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_payments_invoice
        FOREIGN KEY (invoice_id) REFERENCES billing.invoices (id)
);

CREATE TABLE IF NOT EXISTS billing.invoice_items (
    id BIGINT PRIMARY KEY,
    invoice_id BIGINT NOT NULL,
    item_name TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price NUMERIC(12,2) NOT NULL,
    vat_rate NUMERIC(5,2),
    CONSTRAINT fk_items_invoice
        FOREIGN KEY (invoice_id) REFERENCES billing.invoices (id)
);

CREATE TABLE IF NOT EXISTS billing.refunds (
    id BIGINT PRIMARY KEY,
    payment_id BIGINT NOT NULL,
    refund_amount NUMERIC(12,2) NOT NULL,
    refund_reason TEXT,
    refunded_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_refunds_payment
        FOREIGN KEY (payment_id) REFERENCES billing.payments (id)
);

INSERT INTO billing.invoices (id, customer_name, total_amount, issued_at, due_at, paid)
VALUES
    (1, 'North Wind LLC', 150000.00, '2025-01-01', '2025-01-15', TRUE),
    (2, 'Blue River Ltd', 42000.00, '2025-01-05', '2025-01-20', FALSE),
    (3, 'Green Field JSC', 78000.00, '2025-01-08', '2025-01-25', TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO billing.payments (id, invoice_id, paid_amount, payment_method, paid_at)
VALUES
    (1, 1, 150000.00, 'bank_transfer', '2025-01-10 09:00:00'),
    (2, 3, 78000.00, 'card', '2025-01-20 18:15:00')
ON CONFLICT (id) DO NOTHING;

INSERT INTO billing.invoice_items (id, invoice_id, item_name, quantity, unit_price, vat_rate)
VALUES
    (1, 1, 'Annual platform license', 1, 120000.00, 20.00),
    (2, 1, 'Premium support', 1, 30000.00, 20.00),
    (3, 2, 'Consulting hours', 12, 3500.00, 20.00),
    (4, 3, 'Custom integration', 1, 78000.00, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO billing.refunds (id, payment_id, refund_amount, refund_reason, refunded_at)
VALUES
    (1, 2, 8000.00, 'Partial cancellation', '2025-01-25 12:10:00')
ON CONFLICT (id) DO NOTHING;

CREATE OR REPLACE VIEW billing.invoice_summary AS
SELECT
    i.id,
    i.customer_name,
    i.total_amount,
    i.paid,
    COUNT(ii.id) AS item_count,
    COALESCE(SUM(r.refund_amount), 0) AS refunded_amount
FROM billing.invoices i
LEFT JOIN billing.invoice_items ii ON ii.invoice_id = i.id
LEFT JOIN billing.payments p ON p.invoice_id = i.id
LEFT JOIN billing.refunds r ON r.payment_id = p.id
GROUP BY i.id, i.customer_name, i.total_amount, i.paid;
