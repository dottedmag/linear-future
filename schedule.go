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

type scheduleKind int

const (
	scheduleDaily scheduleKind = iota
	scheduleWeekday
	scheduleDayOfMonth
	scheduleLastDayOfMonth
	scheduleMonthDay
	scheduleMonthLast
	scheduleAt
	scheduleMalformed
)

type schedule struct {
	kind    scheduleKind
	weekday time.Weekday // scheduleWeekday
	day     int          // scheduleDayOfMonth, scheduleMonthDay
	month   time.Month   // scheduleMonthDay, scheduleMonthLast
	date    time.Time    // scheduleAt
	raw     string       // scheduleMalformed
}

func (s schedule) matches(date time.Time) bool {
	switch s.kind {
	case scheduleDaily:
		return true
	case scheduleWeekday:
		return date.Weekday() == s.weekday
	case scheduleDayOfMonth:
		return date.Day() == s.day
	case scheduleLastDayOfMonth:
		return date.Day() == lastDayOfMonth(date)
	case scheduleMonthDay:
		return date.Month() == s.month && date.Day() == s.day
	case scheduleMonthLast:
		return date.Month() == s.month && date.Day() == lastDayOfMonth(date)
	case scheduleAt:
		return date.Equal(s.date)
	default:
		return false
	}
}

// parseSchedules extracts all valid schedule entries from a template description.
func parseSchedules(description string) []schedule {
	var schedules []schedule
	for _, line := range strings.Split(description, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)

		if after, ok := strings.CutPrefix(lower, "recurrence:"); ok {
			schedules = append(schedules, parseRecurrence(strings.TrimSpace(after), line))
		} else if after, ok := strings.CutPrefix(lower, "at:"); ok {
			schedules = append(schedules, parseAt(strings.TrimSpace(after), line))
		}
	}
	return schedules
}

func parseRecurrence(value string, rawLine string) schedule {
	lower := strings.ToLower(strings.TrimSpace(value))

	if lower == "daily" {
		return schedule{kind: scheduleDaily}
	}

	if wd, ok := weekdayMap[lower]; ok {
		return schedule{kind: scheduleWeekday, weekday: wd}
	}

	if lower == "last" {
		return schedule{kind: scheduleLastDayOfMonth}
	}

	if day, err := strconv.Atoi(lower); err == nil && day >= 1 && day <= 31 {
		return schedule{kind: scheduleDayOfMonth, day: day}
	}

	parts := strings.Fields(lower)
	if len(parts) == 2 {
		month, ok := monthMap[parts[0]]
		if ok {
			if parts[1] == "last" {
				return schedule{kind: scheduleMonthLast, month: month}
			}
			if day, err := strconv.Atoi(parts[1]); err == nil && day >= 1 && day <= 31 {
				return schedule{kind: scheduleMonthDay, month: month, day: day}
			}
		}
	}

	return schedule{kind: scheduleMalformed, raw: rawLine}
}

func parseAt(value string, rawLine string) schedule {
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return schedule{kind: scheduleMalformed, raw: rawLine}
	}
	return schedule{kind: scheduleAt, date: t}
}

func templateMatchesSchedule(description string, date time.Time) bool {
	for _, s := range parseSchedules(description) {
		if s.matches(date) {
			return true
		}
	}
	return false
}

func lastDayOfMonth(date time.Time) int {
	nextMonth := time.Date(date.Year(), date.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return nextMonth.AddDate(0, 0, -1).Day()
}
