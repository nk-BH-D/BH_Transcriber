package main

import (
	"testing"
	"time"
)

func TestNextTuesdayAtSix(t *testing.T) {
	location := time.FixedZone("test", 3*60*60)
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "before Tuesday run",
			now:  time.Date(2026, time.October, 19, 12, 0, 0, 0, location),
			want: time.Date(2026, time.October, 20, 6, 0, 0, 0, location),
		},
		{
			name: "Tuesday before run",
			now:  time.Date(2026, time.October, 20, 5, 59, 0, 0, location),
			want: time.Date(2026, time.October, 20, 6, 0, 0, 0, location),
		},
		{
			name: "Tuesday at run time",
			now:  time.Date(2026, time.October, 20, 6, 0, 0, 0, location),
			want: time.Date(2026, time.October, 27, 6, 0, 0, 0, location),
		},
		{
			name: "after Tuesday run",
			now:  time.Date(2026, time.October, 21, 8, 0, 0, 0, location),
			want: time.Date(2026, time.October, 27, 6, 0, 0, 0, location),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := nextTuesdayAtTen(test.now); !got.Equal(test.want) {
				t.Errorf("nextTuesdayAtSix(%v) = %v, want %v", test.now, got, test.want)
			}
		})
	}
}
