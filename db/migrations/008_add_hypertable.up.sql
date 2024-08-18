CREATE TABLE IF NOT EXISTS stars_history_hyper (
    repository_id INT REFERENCES repositories(id),
    date DATE NOT NULL,
    star_count INT NOT NULL,
    PRIMARY KEY (repository_id, date)
);

SELECT create_hypertable('stars_history_hyper', 'date');
