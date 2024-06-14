ALTER TABLE repositories
ADD COLUMN primary_language TEXT;

CREATE INDEX IF NOT EXISTS idx_repositories_primary_language ON repositories (primary_language);
