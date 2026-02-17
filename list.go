package main

import (
	"fmt"
	"os"
	"time"
)

func runList(token string) int {
	q := q{token}
	templates, err := getTemplates(q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list templates: %v\n", err)
		return 1
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for _, t := range templates {
		fmt.Printf("%s\n", t.name)
		if t.issueTitle != "" {
			fmt.Printf("  Issue: %s\n", t.issueTitle)
		}

		schedules := parseSchedules(t.description)
		if len(schedules) == 0 {
			fmt.Println("  **NO SCHEDULE**")
		} else {
			for _, s := range schedules {
				fmt.Printf("  %s\n", formatSchedule(s))
			}
			dates := nextTriggerDates(schedules, today, 365, 5)
			if len(dates) > 0 {
				fmt.Print("  Upcoming:")
				for _, d := range dates {
					fmt.Printf(" %s", d.Format("2006-01-02"))
				}
				fmt.Println()
			}
		}

		if len(t.subIssueTitles) > 0 {
			fmt.Println("  Sub-issues:")
			for _, title := range t.subIssueTitles {
				fmt.Printf("    %s\n", title)
			}
			problems := validateSubIssuePrefixes(t.subIssueTitles)
			for _, p := range problems {
				fmt.Printf("  **INVALID**: %s: %s\n", p.title, p.problem)
			}
		}
	}
	return 0
}

func formatSchedule(s schedule) string {
	switch s.kind {
	case scheduleDaily:
		return "Daily"
	case scheduleWeekday:
		return fmt.Sprintf("Every %s", s.weekday)
	case scheduleDayOfMonth:
		return fmt.Sprintf("Day %d of every month", s.day)
	case scheduleLastDayOfMonth:
		return "Last day of every month"
	case scheduleMonthDay:
		return fmt.Sprintf("%s %d", s.month, s.day)
	case scheduleMonthLast:
		return fmt.Sprintf("Last day of %s", s.month)
	case scheduleAt:
		return fmt.Sprintf("Once on %s", s.date.Format("2006-01-02"))
	case scheduleMalformed:
		return fmt.Sprintf("**MALFORMED**: %s", s.raw)
	default:
		return "Unknown"
	}
}

// nextTriggerDates returns up to maxDates matching dates within the next days from from.
func nextTriggerDates(schedules []schedule, from time.Time, days int, maxDates int) []time.Time {
	var dates []time.Time
	for i := range days {
		if len(dates) >= maxDates {
			break
		}
		d := from.AddDate(0, 0, i)
		for _, s := range schedules {
			if s.matches(d) {
				dates = append(dates, d)
				break
			}
		}
	}
	return dates
}
