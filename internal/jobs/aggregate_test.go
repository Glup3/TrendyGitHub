package jobs

import (
	"reflect"
	"testing"
	"time"
)

func TestAggregateStars(t *testing.T) {
	tests := []struct {
		name     string
		input    []time.Time
		expected map[time.Time]int
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
