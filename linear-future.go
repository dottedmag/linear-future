package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

var titleDateRx = regexp.MustCompile(`^@(\d{4}-\d{2}-\d{2})\s+`)

func extractDateFromTitle(title string) (_ time.Time, retFound bool) {
	m := titleDateRx.FindStringSubmatch(title)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", m[1])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func handleIssues(token string, teamName string, labelName string) int {
	q := q{token}
	teamID, labelID, err := getTeamAndLabel(q, teamName, labelName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve team or label: %v\n", err)
		return 1
	}
	fmt.Printf("Team %q resolved to ID %s\n", teamName, teamID)
	fmt.Printf("Label %q resolved to ID %s\n", labelName, labelID)

	today := time.Now().UTC().Truncate(24 * time.Hour)

	isss, err := getTeamIssues(q, teamID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get team issues: %v\n", err)
		return 1
	}
	for _, iss := range isss {
		if !slices.Contains(iss.labels, labelID) {
			continue
		}
		if t, found := extractDateFromTitle(iss.title); found {
			if !t.After(today) {
				fmt.Printf("Issue %q is no longer in the future, removing the label\n", iss.title)
				if err := removeLabel(q, iss.id, labelID); err != nil {
					fmt.Fprintf(os.Stderr, "failed to remove label from issue %q: %v\n", iss.title, err)
					return 1
				}
			}
		} else {
			if !strings.HasPrefix(iss.title, "@? ") {
				fmt.Printf("Issue %q has malformed/missing date, highlighting it and removing the label\n", iss.title)
				if err := updateTitle(q, iss.id, "@? "+iss.title); err != nil {
					fmt.Fprintf(os.Stderr, "failed to update title of issue %q: %v\n", iss.title, err)
					return 1
				}
				if err := removeLabel(q, iss.id, labelID); err != nil {
					fmt.Fprintf(os.Stderr, "failed to remove label from issue %q: %v\n", iss.title, err)
					return 1
				}
			}
		}
	}
	return 0
}

func realMain() int {
	token := os.Getenv("LINEAR_API_KEY")
	if token == "" || len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: LINEAR_API_KEY=lin_api_... linear-future <team name>\n")
		return 2
	}
	return handleIssues(token, os.Args[1], "Later")
}

func main() {
	os.Exit(realMain())
}
