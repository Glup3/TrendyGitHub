CREATE TABLE settings (
    id SERIAL PRIMARY KEY,
    current_max_star_count INT NOT NULL
);

INSERT INTO settings(current_max_star_count) VALUES (1000000);
