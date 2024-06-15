CREATE TABLE IF NOT EXISTS repositories (
    id SERIAL PRIMARY KEY,
    github_id VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    name_with_owner VARCHAR(255) NOT NULL UNIQUE,
    star_count INTEGER NOT NULL,
    fork_count INTEGER NOT NULL,
    primary_language TEXT,
    languages TEXT[] NOT NULL
);

CREATE TABLE IF NOT EXISTS stars_history (
    id BIGSERIAL PRIMARY KEY,
    repository_id INT REFERENCES repositories(id),
    star_count INT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    id INT PRIMARY KEY NOT NULL,
    current_max_star_count INT NOT NULL DEFAULT 1000000,
    min_star_count INT NOT NULL DEFAULT 50,
    timeout_seconds_prevent INT NOT NULL DEFAULT 40,
    timeout_seconds_exceeded INT NOT NULL DEFAULT 60,
    timeout_max_units INT NOT NULL DEFAULT 60,
    enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_stars_history_created_at ON stars_history(created_at);
CREATE INDEX IF NOT EXISTS idx_stars_history_repository_id ON stars_history(repository_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_stars_history_repository_created ON stars_history(repository_id, created_at);
CREATE INDEX IF NOT EXISTS idx_repositories_languages_gin ON repositories USING GIN (languages);
CREATE INDEX IF NOT EXISTS idx_repositories_primary_language ON repositories (primary_language);

INSERT INTO settings(id) VALUES (1);
