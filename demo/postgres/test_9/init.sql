CREATE SCHEMA IF NOT EXISTS education;

CREATE TABLE IF NOT EXISTS education.courses (
    id BIGINT PRIMARY KEY,
    course_name TEXT NOT NULL,
    category TEXT,
    published BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS education.enrollments (
    id BIGINT PRIMARY KEY,
    student_name TEXT NOT NULL,
    course_id BIGINT NOT NULL,
    enrolled_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    CONSTRAINT fk_enrollments_course
        FOREIGN KEY (course_id) REFERENCES education.courses (id)
);

CREATE TABLE IF NOT EXISTS education.lessons (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    lesson_title TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL,
    published_at TIMESTAMP,
    CONSTRAINT fk_lessons_course
        FOREIGN KEY (course_id) REFERENCES education.courses (id)
);

CREATE TABLE IF NOT EXISTS education.course_reviews (
    id BIGINT PRIMARY KEY,
    course_id BIGINT NOT NULL,
    reviewer_name TEXT,
    rating INTEGER NOT NULL,
    review_text TEXT,
    reviewed_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_reviews_course
        FOREIGN KEY (course_id) REFERENCES education.courses (id)
);

INSERT INTO education.courses (id, course_name, category, published)
VALUES
    (1, 'Go Basics', 'backend', TRUE),
    (2, 'SQL Deep Dive', 'database', TRUE),
    (3, 'System Design', NULL, FALSE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO education.enrollments (id, student_name, course_id, enrolled_at, completed_at)
VALUES
    (1, 'Alex Orlov', 1, '2025-01-03 10:00:00', '2025-02-01 10:00:00'),
    (2, 'Nina Frolova', 2, '2025-01-07 15:30:00', NULL),
    (3, 'Sergey Ivanov', 1, '2025-01-10 09:15:00', NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO education.lessons (id, course_id, lesson_title, duration_minutes, published_at)
VALUES
    (1, 1, 'Variables and Types', 45, '2025-01-01 09:00:00'),
    (2, 1, 'Interfaces and Methods', 60, '2025-01-02 09:00:00'),
    (3, 2, 'Joins and Aggregations', 70, '2025-01-03 09:00:00'),
    (4, 3, 'Context and Tradeoffs', 55, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO education.course_reviews (id, course_id, reviewer_name, rating, review_text, reviewed_at)
VALUES
    (1, 1, 'Alex Orlov', 5, 'Clear and practical', '2025-02-02 12:00:00'),
    (2, 2, 'Nina Frolova', 4, 'Great SQL examples', '2025-02-10 17:30:00'),
    (3, 1, NULL, 5, NULL, '2025-02-15 08:45:00')
ON CONFLICT (id) DO NOTHING;

CREATE OR REPLACE VIEW education.course_progress AS
SELECT
    c.id,
    c.course_name,
    COUNT(DISTINCT e.id) AS enrollment_count,
    COUNT(DISTINCT l.id) AS lessons_count,
    AVG(r.rating)::NUMERIC(10,2) AS avg_rating
FROM education.courses c
LEFT JOIN education.enrollments e ON e.course_id = c.id
LEFT JOIN education.lessons l ON l.course_id = c.id
LEFT JOIN education.course_reviews r ON r.course_id = c.id
GROUP BY c.id, c.course_name;
