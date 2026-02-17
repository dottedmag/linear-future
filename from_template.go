package main

import (
	"fmt"
	"time"
)

func createScheduledTeamIssues(token string, teamName string) error {
	q := q{token}
	teamID, err := getTeamID(q, teamName)
	if err != nil {
		return fmt.Errorf("failed to resolve team: %w", err)
	}
	fmt.Printf("Team %q resolved to ID %s\n", teamName, teamID)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	if err := createFromDueTemplates(q, teamID, today); err != nil {
		return fmt.Errorf("failed to create issues from templates: %w", err)
	}
	return nil
}

func createFromDueTemplates(q q, teamID string, today time.Time) error {
	templates, err := getTemplates(q)
	if err != nil {
		return err
	}

	var dueTemplates []issueTemplate
	for _, tmpl := range templates {
		if tmpl.teamID != teamID {
			continue
		}
		if templateMatchesSchedule(tmpl.description, today) {
			fmt.Printf("Template %q is due today\n", tmpl.name)
			dueTemplates = append(dueTemplates, tmpl)
		} else {
			fmt.Printf("Template %q is not due today\n", tmpl.name)
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
		issueID, err := createIssueFromTemplate(q, tmpl.id, teamID)
		if err != nil {
			return err
		}
		if err := setupSubIssueDependencies(q, issueID); err != nil {
			return fmt.Errorf("setting up sub-issue dependencies for template %q: %w", tmpl.name, err)
		}
	}
	return nil
}
