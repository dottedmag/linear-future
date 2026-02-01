package main

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var recurrenceTokenRx = regexp.MustCompile(`@([A-Za-z0-9,-]+)`)

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

func templateMatchesRecurrence(name string, date time.Time) bool {
	matches := false
	found := false
	for _, token := range recurrenceTokenRx.FindAllStringSubmatch(name, -1) {
		raw := strings.Trim(token[1], ",")
		if raw == "" {
			continue
		}
		match, ok := matchRecurrenceToken(raw, date)
		if ok {
			found = true
			if match {
				matches = true
			}
		}
	}
	return found && matches
}

func matchRecurrenceToken(token string, date time.Time) (bool, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, false
	}
	lower := strings.ToLower(token)
	if lower == "daily" {
		return true, true
	}

	if weekdays, ok := parseWeekdayList(lower); ok {
		return containsWeekday(weekdays, date.Weekday()), true
	}

	if isNumericStart(lower) {
		if days, ok := parseDayList(lower); ok {
			return matchesAnyDayOfMonth(date, days), true
		}
	}

	if months, day, ok := parseMonthDaySpec(lower); ok {
		if !containsMonth(months, date.Month()) {
			return false, true
		}
		return matchesDayOfMonth(date, day), true
	}

	return false, false
}

func parseWeekdayList(token string) ([]time.Weekday, bool) {
	parts := strings.Split(token, ",")
	if len(parts) == 0 {
		return nil, false
	}
	days := make([]time.Weekday, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		day, ok := weekdayMap[part]
		if !ok {
			return nil, false
		}
		days = append(days, day)
	}
	return days, true
}

func parseDayList(token string) ([]int, bool) {
	parts := strings.Split(token, ",")
	if len(parts) == 0 {
		return nil, false
	}
	days := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		day, ok := parseDaySpec(part)
		if !ok {
			return nil, false
		}
		days = append(days, day)
	}
	return days, true
}

func parseDaySpec(token string) (int, bool) {
	day, err := strconv.Atoi(token)
	if err != nil {
		return 0, false
	}
	if day == -1 {
		return -1, true
	}
	if day < 1 || day > 31 {
		return 0, false
	}
	return day, true
}

func parseMonthDaySpec(token string) ([]time.Month, int, bool) {
	parts := strings.Split(token, ",")
	if len(parts) == 0 {
		return nil, 0, false
	}
	var months []time.Month
	daySet := 0
	daySetOK := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, 0, false
		}
		letters, digits, ok := splitAlphaDigits(part)
		if !ok {
			return nil, 0, false
		}
		month, ok := monthMap[letters]
		if !ok {
			return nil, 0, false
		}
		months = append(months, month)
		if digits != "" {
			day, ok := parseDaySpec(digits)
			if !ok {
				return nil, 0, false
			}
			if daySetOK && daySet != day {
				return nil, 0, false
			}
			daySet = day
			daySetOK = true
		}
	}
	if !daySetOK {
		return nil, 0, false
	}
	return months, daySet, true
}

func splitAlphaDigits(part string) (string, string, bool) {
	if part == "" {
		return "", "", false
	}
	i := 0
	for i < len(part) && part[i] >= 'a' && part[i] <= 'z' {
		i++
	}
	if i == 0 {
		return "", "", false
	}
	letters := part[:i]
	digits := part[i:]
	if digits == "" {
		return letters, "", true
	}
	start := 0
	if digits[0] == '-' {
		if len(digits) == 1 {
			return "", "", false
		}
		start = 1
	}
	for j := start; j < len(digits); j++ {
		if digits[j] < '0' || digits[j] > '9' {
			return "", "", false
		}
	}
	return letters, digits, true
}

func isNumericStart(token string) bool {
	if token == "" {
		return false
	}
	if token[0] == '-' {
		return len(token) > 1 && token[1] >= '0' && token[1] <= '9'
	}
	return token[0] >= '0' && token[0] <= '9'
}

func containsWeekday(days []time.Weekday, day time.Weekday) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}

func containsMonth(months []time.Month, month time.Month) bool {
	for _, m := range months {
		if m == month {
			return true
		}
	}
	return false
}

func matchesAnyDayOfMonth(date time.Time, days []int) bool {
	for _, day := range days {
		if matchesDayOfMonth(date, day) {
			return true
		}
	}
	return false
}

func matchesDayOfMonth(date time.Time, day int) bool {
	if day == -1 {
		return date.Day() == lastDayOfMonth(date)
	}
	return date.Day() == day
}

func lastDayOfMonth(date time.Time) int {
	nextMonth := time.Date(date.Year(), date.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return nextMonth.AddDate(0, 0, -1).Day()
}
