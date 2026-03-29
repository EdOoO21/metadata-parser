CREATE SCHEMA IF NOT EXISTS support;

CREATE TABLE IF NOT EXISTS support.tickets (
    id BIGINT PRIMARY KEY,
    requester_email TEXT,
    title TEXT NOT NULL,
    priority TEXT NOT NULL,
    status TEXT NOT NULL,
    opened_at TIMESTAMP NOT NULL,
    closed_at TIMESTAMP
);

COMMENT ON TABLE support.tickets IS 'Support tickets';
COMMENT ON COLUMN support.tickets.requester_email IS 'Requester email';

INSERT INTO support.tickets (id, requester_email, title, priority, status, opened_at, closed_at)
VALUES
    (1, 'client1@example.com', 'Cannot login', 'high', 'resolved', '2025-02-10 08:00:00', '2025-02-10 10:00:00'),
    (2, 'client2@example.com', 'Billing mismatch', 'medium', 'open', '2025-02-11 09:15:00', NULL),
    (3, NULL, 'General question', 'low', 'open', '2025-02-12 13:40:00', NULL)
ON CONFLICT (id) DO NOTHING;

CREATE OR REPLACE VIEW support.open_tickets AS
SELECT
    id,
    requester_email,
    priority,
    opened_at
FROM support.tickets
WHERE status = 'open';
