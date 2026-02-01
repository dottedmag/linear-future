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

func handleIssues(token string, teamNames []string, labelName string) int {
	retCode := 0
	for _, teamName := range teamNames {
		if err := handleTeamIssues(token, teamName, labelName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			retCode = 1
		}
	}
	return retCode
}

func handleTeamIssues(token string, teamName string, labelName string) error {
	q := q{token}
	teamID, labelID, err := getTeamAndLabel(q, teamName, labelName)
	if err != nil {
		return fmt.Errorf("failed to resolve team or label: %w", err)
	}
	fmt.Printf("Team %q resolved to ID %s\n", teamName, teamID)
	fmt.Printf("Label %q resolved to ID %s\n", labelName, labelID)

	today := time.Now().UTC().Truncate(24 * time.Hour)

	isss, err := getTeamIssues(q, teamID)
	if err != nil {
		return fmt.Errorf("failed to get team issues: %w", err)
	}
	for _, iss := range isss {
		if !slices.Contains(iss.labels, labelID) {
			continue
		}
		if t, found := extractDateFromTitle(iss.title); found {
			if !t.After(today) {
				fmt.Printf("Issue %q is no longer in the future, removing the label\n", iss.title)
				if err := removeLabel(q, iss.id, labelID); err != nil {
					return fmt.Errorf("failed to remove label from issue %q: %w", iss.title, err)
				}
			}
		} else {
			if !strings.HasPrefix(iss.title, "@? ") {
				fmt.Printf("Issue %q has malformed/missing date, highlighting it and removing the label\n", iss.title)
				if err := updateTitle(q, iss.id, "@? "+iss.title); err != nil {
					return fmt.Errorf("failed to update title of issue %q: %w", iss.title, err)
				}
				if err := removeLabel(q, iss.id, labelID); err != nil {
					return fmt.Errorf("failed to remove label from issue %q: %w", iss.title, err)
				}
			}
		}
	}
	if err := createRecurringIssuesFromTemplates(q, teamID, today); err != nil {
		return fmt.Errorf("failed to create recurring issues from templates: %w", err)
	}
	return nil
}

func createRecurringIssuesFromTemplates(q q, teamID string, today time.Time) error {
	templates, err := getTemplates(q)
	if err != nil {
		return err
	}

	var dueTemplates []issueTemplate
	for _, tmpl := range templates {
		if tmpl.teamID != teamID {
			continue
		}
		if templateMatchesRecurrence(tmpl.name, today) {
			dueTemplates = append(dueTemplates, tmpl)
		}
	}
	if len(dueTemplates) == 0 {
		return nil
	}

	createdToday, err := getTemplateCreatedIssuesForDay(q, teamID, today)
	if err != nil {
		return err
	}

	for _, tmpl := range dueTemplates {
		if createdToday[tmpl.id] {
			fmt.Printf("Template %q already created today, skipping\n", tmpl.name)
			continue
		}
		fmt.Printf("Creating issue from template %q\n", tmpl.name)
		if err := createIssueFromTemplate(q, tmpl.id, teamID); err != nil {
			return err
		}
	}
	return nil
}

func realMain() int {
	token := os.Getenv("LINEAR_API_KEY")
	if len(os.Args) == 2 && os.Args[1] == "--list-templates" {
		return runListTemplates(token)
	}
	if token == "" || len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: LINEAR_API_KEY=lin_api_... linear-future <team name>\n")
		return 2
	}
	return handleIssues(token, os.Args[1:], "Later")
}

func main() {
	os.Exit(realMain())
}
