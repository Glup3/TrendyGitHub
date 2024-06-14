DROP INDEX IF EXISTS idx_repositories_primary_language;

ALTER TABLE repositories
DROP COLUMN primary_language;
