package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func getTeamID(q q, teamName string) (string, error) {
	query := `query GetTeam($teamName: String!) {
		teams(filter: {name: {eq: $teamName}}) {
			nodes { id }
		}
	}`
	body, err := q.do(query, map[string]any{"teamName": teamName})
	if err != nil {
		return "", err
	}

	var resp struct {
		Data struct {
			Teams struct {
				Nodes []struct {
					ID string
				}
			}
		}
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if len(resp.Data.Teams.Nodes) == 0 {
		return "", fmt.Errorf("failed to resolve team name %q to an ID: no team found", teamName)
	}

	return resp.Data.Teams.Nodes[0].ID, nil
}

// searchTeamIssues searches for issues in a team whose title contains the given string.
func searchTeamIssues(q q, teamID, titleContains string) ([]subIssue, error) {
	query := `query SearchIssues($teamID: ID!, $title: StringComparator!, $after: String) {
		issues(filter: { team: {id: {eq: $teamID}}, title: $title }, first: 50, after: $after) {
			nodes { id title }
			pageInfo { hasNextPage endCursor }
		}
	}`

	var out []subIssue
	cursor := ""
	for {
		vars := map[string]any{
			"teamID": teamID,
			"title":  map[string]any{"contains": titleContains},
		}
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
						ID    string
						Title string
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
			out = append(out, subIssue{id: n.ID, title: n.Title})
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

type issueTemplate struct {
	id          string
	name        string
	description string
	teamID      string
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
			description
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
				ID          string
				Name        string
				Description string
				Team        struct {
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
		templates = append(templates, issueTemplate{id: t.ID, name: t.Name, description: t.Description, teamID: t.Team.ID})
	}
	return templates, nil
}

func listTemplatesConnection(q q) ([]issueTemplate, error) {
	query := `query Templates($after: String) {
		templates(first: 50, after: $after) {
			nodes {
				id
				name
				description
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
						ID          string
						Name        string
						Description string
						Team        struct {
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
			templates = append(templates, issueTemplate{id: t.ID, name: t.Name, description: t.Description, teamID: t.Team.ID})
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

type issueTemplateField struct {
	name      string
	selection string
}

// introspectedField represents a field discovered via GraphQL schema introspection.
type introspectedField struct {
	name     string
	leafKind string // after unwrapping NON_NULL/LIST
	leafName string
}

// introspectIssueFields returns all fields on the Issue type via schema introspection.
func introspectIssueFields(q q) ([]introspectedField, error) {
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

	type gqlTypeRef struct {
		Kind   string      `json:"kind"`
		Name   string      `json:"name"`
		OfType *gqlTypeRef `json:"ofType"`
	}
	unwrap := func(ref *gqlTypeRef) *gqlTypeRef {
		for ref != nil && (ref.Kind == "NON_NULL" || ref.Kind == "LIST") {
			ref = ref.OfType
		}
		return ref
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

	var out []introspectedField
	for _, f := range resp.Data.Type.Fields {
		leaf := unwrap(&f.Type)
		if leaf == nil {
			continue
		}
		out = append(out, introspectedField{name: f.Name, leafKind: leaf.Kind, leafName: leaf.Name})
	}
	return out, nil
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

func createIssueFromTemplate(q q, templateID, teamID string) (string, error) {
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
		return "", err
	}

	var resp struct {
		Data struct {
			IssueCreate struct {
				Success bool `json:"success"`
				Issue   struct {
					ID string `json:"id"`
				} `json:"issue"`
			} `json:"issueCreate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if len(resp.Errors) > 0 {
		return "", fmt.Errorf("issueCreate failed: %s", resp.Errors[0].Message)
	}
	if !resp.Data.IssueCreate.Success {
		return "", fmt.Errorf("failed to create issue from template %s", templateID)
	}
	return resp.Data.IssueCreate.Issue.ID, nil
}

// subIssue represents a sub-issue fetched from the API.
type subIssue struct {
	id    string
	title string
}

// getChildIssues fetches all sub-issues (children) of the given parent issue.
func getChildIssues(q q, parentID string) ([]subIssue, error) {
	query := `query GetChildren($issueId: String!) {
		issue(id: $issueId) {
			children(first: 50) {
				nodes {
					id
					title
				}
			}
		}
	}`

	body, err := q.do(query, map[string]any{"issueId": parentID})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Issue struct {
				Children struct {
					Nodes []struct {
						ID    string
						Title string
					}
				}
			}
		}
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	var out []subIssue
	for _, n := range resp.Data.Issue.Children.Nodes {
		out = append(out, subIssue{id: n.ID, title: n.Title})
	}
	return out, nil
}

// createBlocksRelation creates a "blocks" relation: blocker blocks blocked.
func createBlocksRelation(q q, blockerID, blockedID string) error {
	mutation := `mutation CreateRelation($input: IssueRelationCreateInput!) {
		issueRelationCreate(input: $input) {
			success
		}
	}`

	input := map[string]any{
		"issueId":        blockerID,
		"relatedIssueId": blockedID,
		"type":           "blocks",
	}

	body, err := q.do(mutation, map[string]any{"input": input})
	if err != nil {
		return err
	}

	var resp struct {
		Data struct {
			IssueRelationCreate struct {
				Success bool `json:"success"`
			} `json:"issueRelationCreate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("issueRelationCreate failed: %s", resp.Errors[0].Message)
	}
	if !resp.Data.IssueRelationCreate.Success {
		return fmt.Errorf("issueRelationCreate returned success=false")
	}
	return nil
}
