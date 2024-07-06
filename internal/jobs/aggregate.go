package jobs

import "time"

func aggregateStars(dates []time.Time) map[time.Time]int {
	starsByDate := make(map[time.Time]int)

	for _, date := range dates {
		dateMidnight := date.Truncate(24 * time.Hour)
		starsByDate[dateMidnight]++
	}

	return starsByDate
}
