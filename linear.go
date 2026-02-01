package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type q struct {
	token string
}

func (q q) do(query string, variables map[string]any) ([]byte, error) {

	reqBody, err := json.Marshal(struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables,omitempty"` // NB! need omitempty, so not map[string]any
	}{Query: query, Variables: variables})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://api.linear.app/graphql", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", q.token) // NB! no "bearer"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 response: %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (q q) doWithStatus(query string, variables map[string]any) ([]byte, int, error) {
	reqBody, err := json.Marshal(struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables,omitempty"` // NB! need omitempty, so not map[string]any
	}{Query: query, Variables: variables})
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest("POST", "https://api.linear.app/graphql", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", q.token) // NB! no "bearer"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	return body, resp.StatusCode, nil
}

func getTeamAndLabel(q q, teamName, labelName string) (retTeamID string, retLabelID string, _ error) {
	query := `query GetTeamDataWithIssueLabels($teamName: String!, $labelName: String!) {
            teams(filter: {name: {eq: $teamName}}) {
                nodes {
                    id
                    labels(filter: {name: {eq: $labelName}}, first: 1) {
                        nodes {
                            id
                        }
                    }
                }
            }
        }`
	body, err := q.do(query, map[string]any{"teamName": teamName, "labelName": labelName})
	if err != nil {
		return "", "", err
	}

	var resp struct {
		Data struct {
			Teams struct {
				Nodes []struct {
					ID     string
					Labels struct {
						Nodes []struct {
							ID string
						}
					}
				}
			}
		}
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", "", err
	}
	if len(resp.Data.Teams.Nodes) == 0 {
		return "", "", fmt.Errorf("failed to resolve team name %q to an ID: no team found", teamName)
	}
	if len(resp.Data.Teams.Nodes[0].Labels.Nodes) == 0 {
		return "", "", fmt.Errorf("failed to label name %q to an ID: no label found", teamName)
	}

	return resp.Data.Teams.Nodes[0].ID, resp.Data.Teams.Nodes[0].Labels.Nodes[0].ID, nil
}

type iss struct {
	id     string
	title  string
	labels []string
}

func getTeamIssues(q q, teamID string) ([]iss, error) {
	query := `query GetIssues($teamID: ID!, $after: String) {
            issues(filter: { team: {id: {eq: $teamID}}}, first: 20, after: $after) {
                        nodes {
                            id
                            title
                            labels {
                                nodes {
                                    id
                                }
                            }
                        }
                        pageInfo {
                            hasNextPage
                            endCursor
                        }
                    }
        }`

	var out []iss

	cursor := ""
	for {
		vars := map[string]any{"teamID": teamID}
		if cursor != "" {
			vars["after"] = cursor
		}
		body, err := q.do(query, vars)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Data struct {
				Issues struct {
					Nodes []struct {
						ID     string
						Title  string
						Labels struct {
							Nodes []struct {
								ID string
							}
						}
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				}
			}
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}

		for _, n := range resp.Data.Issues.Nodes {
			is := iss{id: n.ID, title: n.Title}
			for _, l := range n.Labels.Nodes {
				is.labels = append(is.labels, l.ID)
			}
			out = append(out, is)
		}

		if !resp.Data.Issues.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Data.Issues.PageInfo.EndCursor
	}
}

func updateTitle(q q, issueID, newTitle string) error {
	mutation := `
	mutation IssueUpdate($id: String!, $title: String!) {
		issueUpdate(id: $id, input: {title: $title}) {
			success
		}
	}`
	variables := map[string]any{
		"id":    issueID,
		"title": newTitle,
	}
	body, err := q.do(mutation, variables)
	if err != nil {
		return err
	}

	var resp struct {
		Data struct {
			IssueUpdate struct {
				Success bool `json:"success"`
			} `json:"issueUpdate"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if !resp.Data.IssueUpdate.Success {
		return fmt.Errorf("failed to update issue %s", issueID)
	}
	return nil
}

func removeLabel(q q, issueID, labelID string) error {
	query := `
		mutation ($issueId: String!, $labelId: String!) {
			issueRemoveLabel(id: $issueId, labelId: $labelId) {
				success
			}
		}
	`

	variables := map[string]any{
		"issueId": issueID,
		"labelId": labelID,
	}

	body, err := q.do(query, variables)
	if err != nil {
		return err
	}

	var resp struct {
		Data struct {
			IssueRemoveLabel struct {
				Success bool `json:"success"`
			} `json:"issueRemoveLabel"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}

	if !resp.Data.IssueRemoveLabel.Success {
		return fmt.Errorf("failed to remove label")
	}

	return nil
}

type issueTemplate struct {
	id     string
	name   string
	teamID string
}

func getTemplates(q q) ([]issueTemplate, error) {
	templates, err := listTemplatesLegacy(q)
	if err == nil {
		return templates, nil
	}
	return listTemplatesConnection(q)
}

func listTemplatesLegacy(q q) ([]issueTemplate, error) {
	query := `query Templates {
		templates {
			id
			name
			team {
				id
			}
		}
	}`

	body, err := q.do(query, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Templates []struct {
				ID   string
				Name string
				Team struct {
					ID string
				}
			}
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("templates query failed: %s", resp.Errors[0].Message)
	}

	templates := make([]issueTemplate, 0, len(resp.Data.Templates))
	for _, t := range resp.Data.Templates {
		templates = append(templates, issueTemplate{id: t.ID, name: t.Name, teamID: t.Team.ID})
	}
	return templates, nil
}

func listTemplatesConnection(q q) ([]issueTemplate, error) {
	query := `query Templates($after: String) {
		templates(first: 50, after: $after) {
			nodes {
				id
				name
				team {
					id
				}
			}
			pageInfo {
				hasNextPage
				endCursor
			}
		}
	}`

	var templates []issueTemplate
	cursor := ""
	for {
		vars := map[string]any{}
		if cursor != "" {
			vars["after"] = cursor
		}

		body, err := q.do(query, vars)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Data struct {
				Templates struct {
					Nodes []struct {
						ID   string
						Name string
						Team struct {
							ID string
						}
					}
					PageInfo struct {
						HasNextPage bool
						EndCursor   string
					}
				}
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		if len(resp.Errors) > 0 {
			return nil, fmt.Errorf("templates query failed: %s", resp.Errors[0].Message)
		}

		for _, t := range resp.Data.Templates.Nodes {
			templates = append(templates, issueTemplate{id: t.ID, name: t.Name, teamID: t.Team.ID})
		}

		if !resp.Data.Templates.PageInfo.HasNextPage {
			return templates, nil
		}
		cursor = resp.Data.Templates.PageInfo.EndCursor
	}
}

type unknownFieldError struct {
	field string
	msg   string
}

func (e unknownFieldError) Error() string {
	return e.msg
}

func isUnknownFieldError(err error) bool {
	_, ok := err.(unknownFieldError)
	return ok
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

type issueTemplateField struct {
	name      string
	selection string
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

func getIssueTemplateFields(q q) ([]issueTemplateField, error) {
	query := `query IssueTemplateFields {
		__type(name: "Issue") {
			fields {
				name
				type {
					kind
					name
					ofType {
						kind
						name
						ofType {
							kind
							name
							ofType {
								kind
								name
							}
						}
					}
				}
			}
		}
	}`

	body, err := q.do(query, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Type *struct {
				Fields []struct {
					Name string     `json:"name"`
					Type gqlTypeRef `json:"type"`
				} `json:"fields"`
			} `json:"__type"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("schema introspection failed: %s", resp.Errors[0].Message)
	}
	if resp.Data.Type == nil {
		return nil, fmt.Errorf("schema introspection failed: Issue type missing")
	}

	fields := make([]issueTemplateField, 0)
	for _, field := range resp.Data.Type.Fields {
		lowerName := strings.ToLower(field.Name)
		leaf := unwrapType(&field.Type)
		if leaf == nil {
			continue
		}
		lowerType := strings.ToLower(leaf.Name)
		if !strings.Contains(lowerName, "template") && !strings.Contains(lowerType, "template") {
			continue
		}
		switch leaf.Kind {
		case "OBJECT":
			fields = append(fields, issueTemplateField{
				name:      field.Name,
				selection: fmt.Sprintf("%s { id }", field.Name),
			})
		case "SCALAR":
			if leaf.Name == "ID" || leaf.Name == "String" {
				fields = append(fields, issueTemplateField{
					name:      field.Name,
					selection: field.Name,
				})
			}
		}
	}

	sort.Slice(fields, func(i, j int) bool {
		return templateFieldPriority(fields[i].name) < templateFieldPriority(fields[j].name)
	})

	return fields, nil
}

type gqlTypeRef struct {
	Kind   string      `json:"kind"`
	Name   string      `json:"name"`
	OfType *gqlTypeRef `json:"ofType"`
}

func unwrapType(ref *gqlTypeRef) *gqlTypeRef {
	for ref != nil && (ref.Kind == "NON_NULL" || ref.Kind == "LIST") {
		ref = ref.OfType
	}
	return ref
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

func getTemplateCreatedIssuesForDayWithField(q q, teamID string, dayStart time.Time, field issueTemplateField) (map[string]bool, error) {
	dayEnd := dayStart.Add(24 * time.Hour)
	created := make(map[string]bool)
	cursor := ""
	for {
		query := fmt.Sprintf(`query IssuesCreatedToday($teamID: ID!, $start: DateTimeOrDuration!, $end: DateTimeOrDuration!, $after: String) {
			issues(filter: { team: {id: {eq: $teamID}}, createdAt: { gte: $start, lt: $end } }, first: 50, after: $after) {
				nodes {
					id
					%s
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}`, field.selection)

		vars := map[string]any{
			"teamID": teamID,
			"start":  dayStart.Format(time.RFC3339),
			"end":    dayEnd.Format(time.RFC3339),
		}
		if cursor != "" {
			vars["after"] = cursor
		}

		body, status, err := q.doWithStatus(query, vars)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Data struct {
				Issues struct {
					Nodes    []map[string]any `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"issues"`
			} `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse issues response (status %d): %w", status, err)
		}
		if len(resp.Errors) > 0 {
			for _, e := range resp.Errors {
				if isUnknownFieldMessage(e.Message) {
					return nil, unknownFieldError{msg: e.Message}
				}
			}
			return nil, fmt.Errorf("issues query failed: %s", resp.Errors[0].Message)
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("issues query failed with status %d: %s", status, string(body))
		}

		for _, n := range resp.Data.Issues.Nodes {
			for _, id := range extractTemplateIDs(n[field.name]) {
				if id != "" {
					created[id] = true
				}
			}
		}

		if !resp.Data.Issues.PageInfo.HasNextPage {
			return created, nil
		}
		cursor = resp.Data.Issues.PageInfo.EndCursor
	}
}

func isUnknownFieldMessage(msg string) bool {
	return strings.Contains(msg, "Cannot query field")
}

func extractTemplateIDs(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		return []string{v}
	case map[string]any:
		if id, ok := v["id"].(string); ok && id != "" {
			return []string{id}
		}
	case []any:
		var ids []string
		for _, item := range v {
			ids = append(ids, extractTemplateIDs(item)...)
		}
		return ids
	}
	return nil
}

func createIssueFromTemplate(q q, templateID, teamID string) error {
	mutation := `
	mutation IssueCreateFromTemplate($templateId: String!, $teamId: String!) {
		issueCreate(input: {templateId: $templateId, teamId: $teamId}) {
			success
			issue {
				id
			}
		}
	}`
	variables := map[string]any{
		"templateId": templateID,
		"teamId":     teamID,
	}
	body, err := q.do(mutation, variables)
	if err != nil {
		return err
	}

	var resp struct {
		Data struct {
			IssueCreate struct {
				Success bool `json:"success"`
			} `json:"issueCreate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("issueCreate failed: %s", resp.Errors[0].Message)
	}
	if !resp.Data.IssueCreate.Success {
		return fmt.Errorf("failed to create issue from template %s", templateID)
	}
	return nil
}
