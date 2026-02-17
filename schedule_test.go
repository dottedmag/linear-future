package main

import (
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestTemplateMatchesSchedule_Daily(t *testing.T) {
	desc := "Some text\nRecurrence: daily\nMore text"
	if !templateMatchesSchedule(desc, date(2025, time.January, 15)) {
		t.Error("daily should match any date")
	}
}

func TestTemplateMatchesSchedule_Weekday(t *testing.T) {
	// 2025-01-13 is a Monday
	mon := date(2025, time.January, 13)
	tue := date(2025, time.January, 14)

	desc := "Recurrence: Mon"
	if !templateMatchesSchedule(desc, mon) {
		t.Error("Mon should match Monday")
	}
	if templateMatchesSchedule(desc, tue) {
		t.Error("Mon should not match Tuesday")
	}
}

func TestTemplateMatchesSchedule_DayNumber(t *testing.T) {
	desc := "Recurrence: 15"
	if !templateMatchesSchedule(desc, date(2025, time.January, 15)) {
		t.Error("15 should match day 15")
	}
	if templateMatchesSchedule(desc, date(2025, time.January, 16)) {
		t.Error("15 should not match day 16")
	}
}

func TestTemplateMatchesSchedule_Last(t *testing.T) {
	desc := "Recurrence: last"
	// Jan has 31 days
	if !templateMatchesSchedule(desc, date(2025, time.January, 31)) {
		t.Error("last should match Jan 31")
	}
	// Feb 2025 has 28 days
	if !templateMatchesSchedule(desc, date(2025, time.February, 28)) {
		t.Error("last should match Feb 28 in non-leap year")
	}
	if templateMatchesSchedule(desc, date(2025, time.January, 30)) {
		t.Error("last should not match Jan 30")
	}
}

func TestTemplateMatchesSchedule_MonthDay(t *testing.T) {
	desc := "Recurrence: Jan 1"
	if !templateMatchesSchedule(desc, date(2025, time.January, 1)) {
		t.Error("Jan 1 should match January 1")
	}
	if templateMatchesSchedule(desc, date(2025, time.February, 1)) {
		t.Error("Jan 1 should not match February 1")
	}
	if templateMatchesSchedule(desc, date(2025, time.January, 2)) {
		t.Error("Jan 1 should not match January 2")
	}
}

func TestTemplateMatchesSchedule_MonthLast(t *testing.T) {
	desc := "Recurrence: Jun last"
	// June has 30 days
	if !templateMatchesSchedule(desc, date(2025, time.June, 30)) {
		t.Error("Jun last should match June 30")
	}
	if templateMatchesSchedule(desc, date(2025, time.June, 29)) {
		t.Error("Jun last should not match June 29")
	}
	if templateMatchesSchedule(desc, date(2025, time.July, 31)) {
		t.Error("Jun last should not match July 31")
	}
}

func TestTemplateMatchesSchedule_MultipleLines(t *testing.T) {
	desc := "Recurrence: Mon\nRecurrence: Fri"
	mon := date(2025, time.January, 13) // Monday
	fri := date(2025, time.January, 17) // Friday
	wed := date(2025, time.January, 15) // Wednesday

	if !templateMatchesSchedule(desc, mon) {
		t.Error("should match Monday")
	}
	if !templateMatchesSchedule(desc, fri) {
		t.Error("should match Friday")
	}
	if templateMatchesSchedule(desc, wed) {
		t.Error("should not match Wednesday")
	}
}

func TestTemplateMatchesSchedule_NoRecurrenceLines(t *testing.T) {
	desc := "Just a regular description"
	if templateMatchesSchedule(desc, date(2025, time.January, 15)) {
		t.Error("no recurrence lines should not match")
	}
}

func TestTemplateMatchesSchedule_CaseInsensitive(t *testing.T) {
	desc := "recurrence: Daily"
	if !templateMatchesSchedule(desc, date(2025, time.January, 15)) {
		t.Error("case-insensitive prefix should work")
	}
}

func TestTemplateMatchesSchedule_EmptyDescription(t *testing.T) {
	if templateMatchesSchedule("", date(2025, time.January, 15)) {
		t.Error("empty description should not match")
	}
}

func TestTemplateMatchesSchedule_At(t *testing.T) {
	desc := "At: 2025-03-15"
	if !templateMatchesSchedule(desc, date(2025, time.March, 15)) {
		t.Error("At should match exact date")
	}
	if templateMatchesSchedule(desc, date(2025, time.March, 16)) {
		t.Error("At should not match different date")
	}
	if templateMatchesSchedule(desc, date(2026, time.March, 15)) {
		t.Error("At should not match different year")
	}
}

func TestTemplateMatchesSchedule_AtCaseInsensitive(t *testing.T) {
	desc := "at: 2025-01-01"
	if !templateMatchesSchedule(desc, date(2025, time.January, 1)) {
		t.Error("at: (lowercase) should work")
	}
}

func TestTemplateMatchesSchedule_AtWithRecurrence(t *testing.T) {
	desc := "At: 2025-06-01\nRecurrence: Mon"
	// 2025-06-01 is a Sunday
	if !templateMatchesSchedule(desc, date(2025, time.June, 1)) {
		t.Error("should match via At line")
	}
	// 2025-06-02 is a Monday
	if !templateMatchesSchedule(desc, date(2025, time.June, 2)) {
		t.Error("should match via Recurrence line")
	}
	if templateMatchesSchedule(desc, date(2025, time.June, 3)) {
		t.Error("should not match Tuesday June 3")
	}
}

func TestTemplateMatchesSchedule_AtInvalidDate(t *testing.T) {
	desc := "At: not-a-date"
	if templateMatchesSchedule(desc, date(2025, time.January, 15)) {
		t.Error("invalid At date should not match")
	}
}
