CREATE TABLE IF NOT EXISTS app_settings (
    key   VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
