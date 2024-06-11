CREATE TABLE repositories (
    id SERIAL PRIMARY KEY,
    github_id VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    url TEXT NOT NULL,
    name_with_owner VARCHAR(255) NOT NULL,
    star_count INTEGER NOT NULL,
    fork_count INTEGER NOT NULL,
    languages TEXT[] NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

