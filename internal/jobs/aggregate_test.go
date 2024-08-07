package jobs

import (
	"reflect"
	"testing"
	"time"
)

func TestAggregateStars(t *testing.T) {
	tests := []struct {
		expected map[time.Time]int
		name     string
		input    []time.Time
	}{
		{
			name:     "Empty input",
			input:    []time.Time{},
			expected: map[time.Time]int{},
		},
		{
			name: "Single date",
			input: []time.Time{
				time.Date(2024, 7, 1, 14, 0, 0, 0, time.UTC),
			},
			expected: map[time.Time]int{
				time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC): 1,
			},
		},
		{
			name: "Multiple dates on the same day",
			input: []time.Time{
				time.Date(2024, 7, 1, 14, 0, 0, 0, time.UTC),
				time.Date(2024, 7, 1, 16, 0, 0, 0, time.UTC),
				time.Date(2024, 7, 1, 18, 0, 0, 0, time.UTC),
			},
			expected: map[time.Time]int{
				time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC): 3,
			},
		},
		{
			name: "Multiple dates on different days",
			input: []time.Time{
				time.Date(2024, 7, 1, 14, 0, 0, 0, time.UTC),
				time.Date(2024, 7, 1, 15, 0, 0, 0, time.UTC),
				time.Date(2024, 7, 1, 16, 0, 0, 0, time.UTC),
				time.Date(2024, 7, 2, 16, 0, 0, 0, time.UTC),
				time.Date(2024, 7, 3, 18, 0, 0, 0, time.UTC),
				time.Date(2024, 7, 3, 12, 0, 0, 0, time.UTC),
			},
			expected: map[time.Time]int{
				time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC): 3,
				time.Date(2024, 7, 2, 0, 0, 0, 0, time.UTC): 1,
				time.Date(2024, 7, 3, 0, 0, 0, 0, time.UTC): 2,
			},
		},
		{
			name: "Dates across midnight",
			input: []time.Time{
				time.Date(2024, 7, 1, 23, 59, 0, 0, time.UTC),
				time.Date(2024, 7, 2, 0, 1, 0, 0, time.UTC),
			},
			expected: map[time.Time]int{
				time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC): 1,
				time.Date(2024, 7, 2, 0, 0, 0, 0, time.UTC): 1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := aggregateStars(test.input)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("unexpected result for %s: got %v, want %v", test.name, result, test.expected)
			}
		})
	}
}

func TestAccumulateStars(t *testing.T) {
	newDate := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}

	tests := []struct {
		starsByDate    map[time.Time]int
		expectedResult map[time.Time]int
		name           string
		baseStarCount  int
	}{
		{
			name:           "Empty map",
			starsByDate:    map[time.Time]int{},
			baseStarCount:  0,
			expectedResult: map[time.Time]int{},
		},
		{
			name: "Single date",
			starsByDate: map[time.Time]int{
				newDate(2024, 7, 1): 5,
			},
			baseStarCount: 0,
			expectedResult: map[time.Time]int{
				newDate(2024, 7, 1): 5,
			},
		},
		{
			name: "Multiple dates",
			starsByDate: map[time.Time]int{
				newDate(2024, 7, 1): 10,
				newDate(2024, 7, 2): 20,
				newDate(2024, 7, 3): 30,
			},
			baseStarCount: 0,
			expectedResult: map[time.Time]int{
				newDate(2024, 7, 1): 10,
				newDate(2024, 7, 2): 30,
				newDate(2024, 7, 3): 60,
			},
		},
		{
			name: "Dates out of order",
			starsByDate: map[time.Time]int{
				newDate(2024, 7, 2): 20,
				newDate(2024, 7, 1): 10,
				newDate(2024, 7, 3): 30,
			},
			baseStarCount: 0,
			expectedResult: map[time.Time]int{
				newDate(2024, 7, 1): 10,
				newDate(2024, 7, 2): 30,
				newDate(2024, 7, 3): 60,
			},
		},
		{
			name: "Non-zero base star count",
			starsByDate: map[time.Time]int{
				newDate(2024, 7, 1): 10,
				newDate(2024, 7, 2): 20,
				newDate(2024, 7, 3): 30,
			},
			baseStarCount: 5,
			expectedResult: map[time.Time]int{
				newDate(2024, 7, 1): 15,
				newDate(2024, 7, 2): 35,
				newDate(2024, 7, 3): 65,
			},
		},
		{
			name: "Multiple stars on the same day",
			starsByDate: map[time.Time]int{
				newDate(2024, 7, 1): 5,
				newDate(2024, 7, 1): 10,
			},
			baseStarCount: 0,
			expectedResult: map[time.Time]int{
				newDate(2024, 7, 1): 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := accumulateStars(tt.starsByDate, tt.baseStarCount)
			if !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("got %v, want %v", result, tt.expectedResult)
			}
		})
	}
}
