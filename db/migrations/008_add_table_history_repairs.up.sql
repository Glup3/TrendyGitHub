CREATE TABLE IF NOT EXISTS history_repairs (
    repository_id INT PRIMARY KEY REFERENCES repositories(id),
    until_date DATE NOT NULL
);

