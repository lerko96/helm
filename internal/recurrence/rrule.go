package recurrence

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseRRule parses a minimal RRULE string supporting FREQ and INTERVAL only.
// Supported FREQ values: DAILY, WEEKLY, MONTHLY.
func ParseRRule(rrule string) (freq string, interval int, err error) {
	interval = 1
	for _, part := range strings.Split(rrule, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.ToUpper(kv[0]) {
		case "FREQ":
			freq = strings.ToUpper(kv[1])
		case "INTERVAL":
			interval, err = strconv.Atoi(kv[1])
			if err != nil {
				return "", 0, fmt.Errorf("invalid INTERVAL: %w", err)
			}
		}
	}
	switch freq {
	case "DAILY", "WEEKLY", "MONTHLY":
	case "":
		return "", 0, fmt.Errorf("FREQ is required")
	default:
		return "", 0, fmt.Errorf("unsupported FREQ: %s (must be DAILY, WEEKLY, or MONTHLY)", freq)
	}
	if interval < 1 {
		return "", 0, fmt.Errorf("INTERVAL must be >= 1")
	}
	return freq, interval, nil
}

// Advance returns t advanced by one period defined by freq and interval.
func Advance(t time.Time, freq string, interval int) time.Time {
	switch freq {
	case "DAILY":
		return t.AddDate(0, 0, interval)
	case "WEEKLY":
		return t.AddDate(0, 0, 7*interval)
	case "MONTHLY":
		return t.AddDate(0, interval, 0)
	default:
		return t.AddDate(0, 0, interval)
	}
}
