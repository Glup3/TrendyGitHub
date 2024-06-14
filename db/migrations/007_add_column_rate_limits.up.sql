ALTER TABLE settings
ADD COLUMN timeout_seconds_prevent INT NOT NULL DEFAULT 40;

ALTER TABLE settings
ADD COLUMN timeout_seconds_exceeded INT NOT NULL DEFAULT 60;

ALTER TABLE settings
ADD COLUMN timeout_max_units INT NOT NULL DEFAULT 60;

