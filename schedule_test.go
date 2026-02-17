package main

import (
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestTemplateMatchesSchedule_Daily(t *testing.T) {
	desc := "Some text|Recurrence: daily|More text"
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.January, 15)))
}

func TestTemplateMatchesSchedule_Weekday(t *testing.T) {
	// 2025-01-13 is a Monday
	mon := date(2025, time.January, 13)
	tue := date(2025, time.January, 14)

	desc := "Recurrence: Mon"
	assert.True(t, templateMatchesSchedule(desc, mon))
	assert.False(t, templateMatchesSchedule(desc, tue))
}

func TestTemplateMatchesSchedule_DayNumber(t *testing.T) {
	desc := "Recurrence: 15"
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.January, 15)))
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.January, 16)))
}

func TestTemplateMatchesSchedule_Last(t *testing.T) {
	desc := "Recurrence: last"
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.January, 31)))
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.February, 28)))
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.January, 30)))
}

func TestTemplateMatchesSchedule_MonthDay(t *testing.T) {
	desc := "Recurrence: Jan 1"
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.January, 1)))
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.February, 1)))
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.January, 2)))
}

func TestTemplateMatchesSchedule_MonthLast(t *testing.T) {
	desc := "Recurrence: Jun last"
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.June, 30)))
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.June, 29)))
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.July, 31)))
}

func TestTemplateMatchesSchedule_MultipleLines(t *testing.T) {
	desc := "Recurrence: Mon|Recurrence: Fri"
	mon := date(2025, time.January, 13) // Monday
	fri := date(2025, time.January, 17) // Friday
	wed := date(2025, time.January, 15) // Wednesday

	assert.True(t, templateMatchesSchedule(desc, mon))
	assert.True(t, templateMatchesSchedule(desc, fri))
	assert.False(t, templateMatchesSchedule(desc, wed))
}

func TestTemplateMatchesSchedule_NoRecurrenceLines(t *testing.T) {
	desc := "Just a regular description"
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.January, 15)))
}

func TestTemplateMatchesSchedule_CaseInsensitive(t *testing.T) {
	desc := "recurrence: Daily"
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.January, 15)))
}

func TestTemplateMatchesSchedule_EmptyDescription(t *testing.T) {
	assert.False(t, templateMatchesSchedule("", date(2025, time.January, 15)))
}

func TestTemplateMatchesSchedule_At(t *testing.T) {
	desc := "At: 2025-03-15"
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.March, 15)))
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.March, 16)))
	assert.False(t, templateMatchesSchedule(desc, date(2026, time.March, 15)))
}

func TestTemplateMatchesSchedule_AtCaseInsensitive(t *testing.T) {
	desc := "at: 2025-01-01"
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.January, 1)))
}

func TestTemplateMatchesSchedule_AtWithRecurrence(t *testing.T) {
	desc := "At: 2025-06-01|Recurrence: Mon"
	// 2025-06-01 is a Sunday
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.June, 1)))
	// 2025-06-02 is a Monday
	assert.True(t, templateMatchesSchedule(desc, date(2025, time.June, 2)))
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.June, 3)))
}

func TestTemplateMatchesSchedule_AtInvalidDate(t *testing.T) {
	desc := "At: not-a-date"
	assert.False(t, templateMatchesSchedule(desc, date(2025, time.January, 15)))
}
