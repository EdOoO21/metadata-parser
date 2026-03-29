CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.sessions (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    device_type TEXT NOT NULL,
    country TEXT,
    is_mobile BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS analytics.events (
    id BIGINT PRIMARY KEY,
    session_id BIGINT NOT NULL,
    event_name TEXT NOT NULL,
    screen_name TEXT,
    event_value NUMERIC(12,2),
    event_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_events_session
        FOREIGN KEY (session_id) REFERENCES analytics.sessions (id)
);

CREATE TABLE IF NOT EXISTS analytics.user_profiles (
    user_id BIGINT PRIMARY KEY,
    country TEXT,
    language TEXT,
    marketing_opt_in BOOLEAN,
    registered_at TIMESTAMP NOT NULL
);

INSERT INTO analytics.sessions (id, user_id, started_at, ended_at, device_type, country, is_mobile)
VALUES
    (1, 1001, '2025-03-01 10:00:00', '2025-03-01 10:45:00', 'ios', 'RU', TRUE),
    (2, 1002, '2025-03-01 11:15:00', NULL, 'web', 'DE', FALSE),
    (3, 1001, '2025-03-02 09:20:00', '2025-03-02 09:55:00', 'android', 'RU', TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO analytics.events (id, session_id, event_name, screen_name, event_value, event_at)
VALUES
    (1, 1, 'app_open', 'home', NULL, '2025-03-01 10:00:03'),
    (2, 1, 'purchase', 'checkout', 1490.00, '2025-03-01 10:30:15'),
    (3, 2, 'page_view', 'pricing', NULL, '2025-03-01 11:18:00'),
    (4, 3, 'search', 'catalog', NULL, '2025-03-02 09:25:00')
ON CONFLICT (id) DO NOTHING;

INSERT INTO analytics.user_profiles (user_id, country, language, marketing_opt_in, registered_at)
VALUES
    (1001, 'RU', 'ru', TRUE, '2024-09-10 08:00:00'),
    (1002, 'DE', 'en', FALSE, '2024-12-01 14:30:00')
ON CONFLICT (user_id) DO NOTHING;

CREATE OR REPLACE VIEW analytics.daily_sessions AS
SELECT
    DATE(started_at) AS day,
    COUNT(*) AS sessions_count
FROM analytics.sessions
GROUP BY DATE(started_at);

CREATE OR REPLACE VIEW analytics.session_overview AS
SELECT
    s.id,
    s.user_id,
    s.device_type,
    s.country,
    COUNT(e.id) AS events_count,
    MAX(e.event_name) FILTER (WHERE e.event_name = 'purchase') AS has_purchase
FROM analytics.sessions s
LEFT JOIN analytics.events e ON e.session_id = s.id
GROUP BY s.id, s.user_id, s.device_type, s.country;
