package main

import (
	"strconv"
	"strings"
	"time"
)

var weekdayMap = map[string]time.Weekday{
	"mon": time.Monday,
	"tue": time.Tuesday,
	"wed": time.Wednesday,
	"thu": time.Thursday,
	"fri": time.Friday,
	"sat": time.Saturday,
	"sun": time.Sunday,
}

var monthMap = map[string]time.Month{
	"jan": time.January,
	"feb": time.February,
	"mar": time.March,
	"apr": time.April,
	"may": time.May,
	"jun": time.June,
	"jul": time.July,
	"aug": time.August,
	"sep": time.September,
	"oct": time.October,
	"nov": time.November,
	"dec": time.December,
}

func templateMatchesSchedule(description string, date time.Time) bool {
	for _, line := range strings.Split(description, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)

		if strings.HasPrefix(lower, "recurrence:") {
			value := strings.TrimSpace(line[len("recurrence:"):])
			if value == "" {
				continue
			}
			matches, valid := parseRecurrenceLine(value, date)
			if valid && matches {
				return true
			}
		}

		if strings.HasPrefix(lower, "at:") {
			value := strings.TrimSpace(line[len("at:"):])
			t, err := time.Parse("2006-01-02", value)
			if err != nil {
				continue
			}
			if t.Equal(date) {
				return true
			}
		}
	}
	return false
}

func parseRecurrenceLine(value string, date time.Time) (matches, valid bool) {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false, false
	}

	if lower == "daily" {
		return true, true
	}

	// Try as weekday
	if wd, ok := weekdayMap[lower]; ok {
		return date.Weekday() == wd, true
	}

	// Try as "last"
	if lower == "last" {
		return date.Day() == lastDayOfMonth(date), true
	}

	// Try as day number
	if day, err := strconv.Atoi(lower); err == nil && day >= 1 && day <= 31 {
		return date.Day() == day, true
	}

	// Try as month + day ("Jan 1", "Jun last")
	parts := strings.Fields(lower)
	if len(parts) == 2 {
		month, ok := monthMap[parts[0]]
		if ok {
			if date.Month() != month {
				return false, true
			}
			if parts[1] == "last" {
				return date.Day() == lastDayOfMonth(date), true
			}
			if day, err := strconv.Atoi(parts[1]); err == nil && day >= 1 && day <= 31 {
				return date.Day() == day, true
			}
		}
	}

	return false, false
}

func lastDayOfMonth(date time.Time) int {
	nextMonth := time.Date(date.Year(), date.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return nextMonth.AddDate(0, 0, -1).Day()
}
