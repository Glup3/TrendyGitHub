CREATE TABLE stars_history (
    id BIGSERIAL PRIMARY KEY,
    repository_id INT REFERENCES repositories(id),
    star_count INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_stars_history_created_at ON stars_history(created_at);
