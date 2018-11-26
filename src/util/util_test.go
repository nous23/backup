package util

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	durationString := []string{"1s", "10m", "01h", "2d", "3w", "030mo"}
	parseResults := []time.Duration{time.Second, time.Minute * 10, time.Hour, time.Hour * 24 * 2,
		time.Hour * 24 * 7 * 3, time.Hour * 24 * 30 * 30}
	for i := 0; i < len(durationString); i++ {
		duration, err := ParseDuration(durationString[i])
		if err != nil {
			t.Error(err)
		}
		if duration != parseResults[i] {
			t.Errorf("parse error: %s --> %v", durationString[i], duration)
		}
	}
}

