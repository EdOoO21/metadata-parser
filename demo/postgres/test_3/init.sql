CREATE SCHEMA IF NOT EXISTS hr;

CREATE TABLE IF NOT EXISTS hr.departments (
    id BIGINT PRIMARY KEY,
    department_name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS hr.employees (
    id BIGINT PRIMARY KEY,
    full_name TEXT NOT NULL,
    department_id BIGINT NOT NULL,
    active BOOLEAN NOT NULL,
    hired_at DATE NOT NULL,
    salary NUMERIC(12,2),
    CONSTRAINT fk_employees_department
        FOREIGN KEY (department_id) REFERENCES hr.departments (id)
);

CREATE TABLE IF NOT EXISTS hr.positions (
    id BIGINT PRIMARY KEY,
    employee_id BIGINT NOT NULL,
    title TEXT NOT NULL,
    grade TEXT,
    manager_id BIGINT,
    assigned_at DATE NOT NULL,
    CONSTRAINT fk_positions_employee
        FOREIGN KEY (employee_id) REFERENCES hr.employees (id)
);

CREATE TABLE IF NOT EXISTS hr.employee_contacts (
    id BIGINT PRIMARY KEY,
    employee_id BIGINT NOT NULL,
    work_email TEXT,
    phone TEXT,
    emergency_contact TEXT,
    emergency_phone TEXT,
    CONSTRAINT fk_contacts_employee
        FOREIGN KEY (employee_id) REFERENCES hr.employees (id)
);

COMMENT ON COLUMN hr.employees.full_name IS 'Employee full name';

INSERT INTO hr.departments (id, department_name)
VALUES
    (1, 'Engineering'),
    (2, 'Finance')
ON CONFLICT (id) DO NOTHING;

INSERT INTO hr.employees (id, full_name, department_id, active, hired_at, salary)
VALUES
    (1, 'Elena Volkova', 1, TRUE, '2021-03-15', 220000.00),
    (2, 'Roman Kozlov', 1, TRUE, '2022-07-01', 185000.00),
    (3, 'Olga Petrova', 2, FALSE, '2020-11-20', NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO hr.positions (id, employee_id, title, grade, manager_id, assigned_at)
VALUES
    (1, 1, 'Engineering Manager', 'M3', NULL, '2022-01-10'),
    (2, 2, 'Backend Engineer', 'IC3', 1, '2022-07-05'),
    (3, 3, 'Financial Analyst', 'IC2', NULL, '2021-01-15')
ON CONFLICT (id) DO NOTHING;

INSERT INTO hr.employee_contacts (id, employee_id, work_email, phone, emergency_contact, emergency_phone)
VALUES
    (1, 1, 'elena.volkova@corp.local', '+74951234567', 'Alexey Volkhov', '+79030000001'),
    (2, 2, 'roman.kozlov@corp.local', NULL, 'Maria Kozlova', '+79030000002'),
    (3, 3, 'olga.petrova@corp.local', '+74957654321', NULL, NULL)
ON CONFLICT (id) DO NOTHING;

CREATE OR REPLACE VIEW hr.active_employees AS
SELECT
    e.id,
    e.full_name,
    d.department_name,
    e.salary
FROM hr.employees e
JOIN hr.departments d ON d.id = e.department_id
WHERE e.active = TRUE;

CREATE OR REPLACE VIEW hr.employee_directory AS
SELECT
    e.id,
    e.full_name,
    d.department_name,
    p.title,
    c.work_email,
    c.phone
FROM hr.employees e
JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions p ON p.employee_id = e.id
LEFT JOIN hr.employee_contacts c ON c.employee_id = e.id;
