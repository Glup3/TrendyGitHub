CREATE TABLE settings (
    id SERIAL PRIMARY KEY,
    cursor_value TEXT NOT NULL
);

INSERT INTO settings(cursor_value) VALUES ('');
