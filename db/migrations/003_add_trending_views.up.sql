CREATE MATERIALIZED VIEW mv_daily_stars AS 
WITH history AS (
	SELECT
		repository_id,
		created_at,
		star_count - lag(star_count) OVER (PARTITION BY repository_id ORDER BY created_at) AS stars_difference
	FROM stars_history
)
SELECT repository_id, sum(stars_difference) AS stars_difference
FROM history
WHERE created_at >= CURRENT_DATE - INTERVAL '1 day'
GROUP BY repository_id
HAVING sum(stars_difference) > 0;


CREATE MATERIALIZED VIEW mv_weekly_stars AS 
WITH history AS (
	SELECT
		repository_id,
		created_at,
		star_count - lag(star_count) OVER (PARTITION BY repository_id ORDER BY created_at) AS stars_difference
	FROM stars_history
)
SELECT repository_id, sum(stars_difference) AS stars_difference
FROM history
WHERE created_at >= CURRENT_DATE - INTERVAL '1 week'
GROUP BY repository_id
HAVING sum(stars_difference) > 0;


CREATE MATERIALIZED VIEW mv_monthly_stars AS 
WITH history AS (
	SELECT
		repository_id,
		created_at,
		star_count - lag(star_count) OVER (PARTITION BY repository_id ORDER BY created_at) AS stars_difference
	FROM stars_history
)
SELECT repository_id, sum(stars_difference) AS stars_difference
FROM history
WHERE created_at >= CURRENT_DATE - INTERVAL '1 month'
GROUP BY repository_id
HAVING sum(stars_difference) > 0;
