package main

import (
	"fmt"
	"os"
	"time"
)

func runTriggerDates(token string) int {
	q := q{token}
	templates, err := getTemplates(q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list templates: %v\n", err)
		return 1
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for _, t := range templates {
		fmt.Printf("%s\n", t.name)
		schedules := parseSchedules(t.description)
		if len(schedules) == 0 {
			fmt.Println("  No schedule")
			continue
		}
		for _, s := range schedules {
			fmt.Printf("  %s\n", s)
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
	return 0
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
