package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

func handleIssues(token string, teamNames []string) int {
	retCode := 0
	for _, teamName := range teamNames {
		if err := handleTeamIssues(token, teamName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			retCode = 1
		}
	}
	return retCode
}

func handleTeamIssues(token string, teamName string) error {
	q := q{token}
	teamID, err := getTeamID(q, teamName)
	if err != nil {
		return fmt.Errorf("failed to resolve team: %w", err)
	}
	fmt.Printf("Team %q resolved to ID %s\n", teamName, teamID)

	today := time.Now().UTC().Truncate(24 * time.Hour)
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
		if templateMatchesSchedule(tmpl.description, today) {
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

func getTemplateCreatedIssuesForDay(q q, teamID string, dayStart time.Time) (map[string]bool, error) {
	fields, err := getIssueTemplateFields(q)
	if err != nil || len(fields) == 0 {
		fields = staticIssueTemplateFields()
	} else {
		fields = appendIssueTemplateFields(fields, staticIssueTemplateFields())
	}

	var lastErr error
	onlyUnknownFields := true
	for _, field := range fields {
		created, err := getTemplateCreatedIssuesForDayWithField(q, teamID, dayStart, field)
		if err == nil {
			return created, nil
		}
		lastErr = err
		if !isUnknownFieldError(err) {
			onlyUnknownFields = false
			return nil, err
		}
	}

	if onlyUnknownFields {
		return nil, fmt.Errorf("no issue template link field found on Issue; cannot enforce one-per-day")
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no issue template link field found on Issue; cannot enforce one-per-day")
}

func getIssueTemplateFields(q q) ([]issueTemplateField, error) {
	rawFields, err := introspectIssueFields(q)
	if err != nil {
		return nil, err
	}

	var fields []issueTemplateField
	for _, f := range rawFields {
		lowerName := strings.ToLower(f.name)
		lowerType := strings.ToLower(f.leafName)
		if !strings.Contains(lowerName, "template") && !strings.Contains(lowerType, "template") {
			continue
		}
		switch f.leafKind {
		case "OBJECT":
			fields = append(fields, issueTemplateField{
				name:      f.name,
				selection: fmt.Sprintf("%s { id }", f.name),
			})
		case "SCALAR":
			if f.leafName == "ID" || f.leafName == "String" {
				fields = append(fields, issueTemplateField{
					name:      f.name,
					selection: f.name,
				})
			}
		}
	}

	sort.Slice(fields, func(i, j int) bool {
		return templateFieldPriority(fields[i].name) < templateFieldPriority(fields[j].name)
	})

	return fields, nil
}

func staticIssueTemplateFields() []issueTemplateField {
	return []issueTemplateField{
		{name: "template", selection: "template { id }"},
		{name: "createdFromTemplate", selection: "createdFromTemplate { id }"},
		{name: "templateId", selection: "templateId"},
		{name: "createdFromTemplateId", selection: "createdFromTemplateId"},
		{name: "createdFrom", selection: "createdFrom { id }"},
	}
}

func appendIssueTemplateFields(fields []issueTemplateField, extra []issueTemplateField) []issueTemplateField {
	seen := make(map[string]bool, len(fields))
	for _, f := range fields {
		seen[f.name] = true
	}
	for _, f := range extra {
		if !seen[f.name] {
			fields = append(fields, f)
		}
	}
	return fields
}

func templateFieldPriority(name string) int {
	switch strings.ToLower(name) {
	case "template":
		return 0
	case "createdfromtemplate":
		return 1
	case "templateid":
		return 2
	case "createdfromtemplateid":
		return 3
	case "createdfrom":
		return 4
	default:
		return 100
	}
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
	return handleIssues(token, os.Args[1:])
}

func main() {
	os.Exit(realMain())
}
