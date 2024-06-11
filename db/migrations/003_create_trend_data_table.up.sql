CREATE TABLE trend_data (
    id BIGSERIAL PRIMARY KEY,
    language VARCHAR(255),
    period VARCHAR(50) NOT NULL,
    repository_id INT REFERENCES repositories(id),
    star_change INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_trend_data_language_period ON trend_data(language, period);

