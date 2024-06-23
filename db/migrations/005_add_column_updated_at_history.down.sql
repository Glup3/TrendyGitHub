DROP TRIGGER IF EXISTS update_updated_at ON stars_history;

DROP FUNCTION IF EXISTS update_timestamp;

ALTER TABLE stars_history
DROP COLUMN IF EXISTS updated_at;

