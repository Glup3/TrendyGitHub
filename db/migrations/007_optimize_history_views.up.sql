CREATE MATERIALIZED VIEW mv_daily_stars_new AS 
WITH history AS (
	SELECT
		repository_id,
		created_at,
		star_count - lag(star_count) OVER (PARTITION BY repository_id ORDER BY created_at) AS stars_difference
	FROM stars_history
)
SELECT repository_id, sum(stars_difference) AS stars_difference
FROM history
WHERE created_at > CURRENT_DATE - INTERVAL '1 day'
GROUP BY repository_id
HAVING sum(stars_difference) > 0;


CREATE MATERIALIZED VIEW mv_weekly_stars_new AS 
WITH history AS (
	SELECT
		repository_id,
		created_at,
		star_count - lag(star_count) OVER (PARTITION BY repository_id ORDER BY created_at) AS stars_difference
	FROM stars_history
)
SELECT repository_id, sum(stars_difference) AS stars_difference
FROM history
WHERE created_at > CURRENT_DATE - INTERVAL '1 week'
GROUP BY repository_id
HAVING sum(stars_difference) > 0;


CREATE MATERIALIZED VIEW mv_monthly_stars_new AS 
WITH history AS (
	SELECT
		repository_id,
		created_at,
		star_count - lag(star_count) OVER (PARTITION BY repository_id ORDER BY created_at) AS stars_difference
	FROM stars_history
)
SELECT repository_id, sum(stars_difference) AS stars_difference
FROM history
WHERE created_at > CURRENT_DATE - INTERVAL '1 month'
GROUP BY repository_id
HAVING sum(stars_difference) > 0;


CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_repoid_mv_daily_stars ON mv_daily_stars_new (repository_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_repoid_mv_weekly_stars ON mv_weekly_stars_new (repository_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_repoid_mv_monthly_stars ON mv_monthly_stars_new (repository_id);



ALTER MATERIALIZED VIEW mv_daily_stars RENAME TO mv_daily_stars_old;
ALTER MATERIALIZED VIEW mv_weekly_stars RENAME TO mv_weekly_stars_old;
ALTER MATERIALIZED VIEW mv_monthly_stars RENAME TO mv_monthly_stars_old;



ALTER MATERIALIZED VIEW mv_daily_stars_new RENAME TO mv_daily_stars;
ALTER MATERIALIZED VIEW mv_weekly_stars_new RENAME TO mv_weekly_stars;
ALTER MATERIALIZED VIEW mv_monthly_stars_new RENAME TO mv_monthly_stars;


DROP MATERIALIZED VIEW IF EXISTS mv_daily_stars_old;
DROP MATERIALIZED VIEW IF EXISTS mv_weekly_stars_old;
DROP MATERIALIZED VIEW IF EXISTS mv_monthly_stars_old;
