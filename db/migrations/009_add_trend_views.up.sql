CREATE MATERIALIZED VIEW if not exists trend_daily as
SELECT
	repository_id,
	first(star_count, date),
	last(star_count, date),
	last(star_count, date) - first(star_count, date) as stars_diff
from stars_history_hyper
WHERE date >= CURRENT_DATE - INTERVAL '2 day'
group by repository_id
having last(star_count, date) - first(star_count, date) > 10
order by stars_diff desc, repository_id;
create unique index if not exists ix_unique_trend_daily_repoid on trend_daily(repository_id);



CREATE MATERIALIZED VIEW if not exists trend_weekly as
SELECT
	repository_id,
	first(star_count, date),
	last(star_count, date),
	last(star_count, date) - first(star_count, date) as stars_diff
from stars_history_hyper
WHERE date >= CURRENT_DATE - INTERVAL '1 week'
group by repository_id
having last(star_count, date) - first(star_count, date) > 10
order by stars_diff desc, repository_id;
create unique index if not exists ix_unique_trend_weekly_repoid on trend_weekly(repository_id);



CREATE MATERIALIZED VIEW if not exists trend_monthly as
SELECT
	repository_id,
	first(star_count, date),
	last(star_count, date),
	last(star_count, date) - first(star_count, date) as stars_diff
from stars_history_hyper
WHERE date >= CURRENT_DATE - INTERVAL '1 month'
group by repository_id
having last(star_count, date) - first(star_count, date) > 10
order by stars_diff desc, repository_id;
create unique index if not exists ix_unique_trend_monthly_repoid on trend_monthly(repository_id);
