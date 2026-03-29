CREATE SCHEMA IF NOT EXISTS pii;

CREATE TABLE IF NOT EXISTS pii.people (
    id BIGINT PRIMARY KEY,
    fio TEXT NOT NULL,
    passport_number TEXT,
    phone TEXT,
    email TEXT,
    birth_date DATE,
    city TEXT
);

COMMENT ON COLUMN pii.people.fio IS 'Full person name';
COMMENT ON COLUMN pii.people.passport_number IS 'Passport number';

INSERT INTO pii.people (id, fio, passport_number, phone, email, birth_date, city)
VALUES
    (1, 'Irina Sokolova', '4510 123456', '+79990000001', 'irina@example.com', '1991-04-12', 'Moscow'),
    (2, 'Dmitry Lebedev', '4511 654321', '+79990000002', NULL, '1988-09-03', 'Tver'),
    (3, 'Maria Egorova', NULL, NULL, 'maria@example.com', NULL, 'Yaroslavl')
ON CONFLICT (id) DO NOTHING;
