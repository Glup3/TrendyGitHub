CREATE INDEX IF NOT EXISTS idx_repositories_languages_gin ON repositories USING GIN (languages);

