package jobs

import (
	"sort"
	"time"
)

func aggregateStars(dates []time.Time) map[time.Time]int {
	starsByDate := make(map[time.Time]int)

	for _, date := range dates {
		dateMidnight := date.Truncate(24 * time.Hour)
		starsByDate[dateMidnight]++
	}

	return starsByDate
}

func accumulateStars(starsByDate map[time.Time]int, baseStarCount int) map[time.Time]int {
	accumulatedStarsByDate := make(map[time.Time]int)

	var sortedDates []time.Time
	for date := range starsByDate {
		sortedDates = append(sortedDates, date)
	}
	sort.Slice(sortedDates, func(i, j int) bool {
		return sortedDates[i].Before(sortedDates[j])
	})

	cumulativeSum := baseStarCount
	for _, date := range sortedDates {
		cumulativeSum += starsByDate[date]
		accumulatedStarsByDate[date] = cumulativeSum
	}

	return accumulatedStarsByDate
}
