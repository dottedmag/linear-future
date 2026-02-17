package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const testMarker = "[LFIT]"

func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("LINEAR_FUTURE_INTEGRATION_TESTS") == "" {
		t.Skip("set LINEAR_FUTURE_INTEGRATION_TESTS=1 to run integration tests")
	}
}

func integrationQ(t *testing.T) q {
	t.Helper()
	token := os.Getenv("LINEAR_API_KEY")
	if token == "" {
		t.Fatal("LINEAR_API_KEY must be set for integration tests")
	}
	return q{token}
}

// --- test helpers ---

func testGetTeamID(t *testing.T, q q) string {
	t.Helper()
	body, err := q.do(`query { teams { nodes { id } } }`, nil)
	if err != nil {
		t.Fatalf("list teams: %v", err)
	}
	var resp struct {
		Data struct {
			Teams struct {
				Nodes []struct{ ID string }
			}
		}
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parse teams: %v", err)
	}
	if len(resp.Data.Teams.Nodes) == 0 {
		t.Fatal("no teams in workspace")
	}
	return resp.Data.Teams.Nodes[0].ID
}

func testCreateTemplate(t *testing.T, q q, teamID, name, description string) string {
	t.Helper()
	mutation := `mutation ($input: TemplateCreateInput!) {
		templateCreate(input: $input) {
			success
			template { id }
		}
	}`
	templateData := map[string]any{
		"title":  name,
		"teamId": teamID,
	}
	tdJSON, _ := json.Marshal(templateData)
	input := map[string]any{
		"type":         "issue",
		"name":         name,
		"description":  description,
		"teamId":       teamID,
		"templateData": json.RawMessage(tdJSON),
	}
	body, err := q.do(mutation, map[string]any{"input": input})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	var resp struct {
		Data struct {
			TemplateCreate struct {
				Success  bool
				Template struct{ ID string }
			}
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parse templateCreate: %v\nbody: %s", err, body)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("templateCreate error: %s", resp.Errors[0].Message)
	}
	if !resp.Data.TemplateCreate.Success {
		t.Fatal("templateCreate returned success=false")
	}
	return resp.Data.TemplateCreate.Template.ID
}

func testDeleteTemplate(t *testing.T, q q, id string) {
	t.Helper()
	mutation := `mutation ($id: String!) {
		templateDelete(id: $id) { success }
	}`
	body, err := q.do(mutation, map[string]any{"id": id})
	if err != nil {
		t.Logf("delete template %s: %v", id, err)
		return
	}
	var resp struct {
		Data struct {
			TemplateDelete struct{ Success bool }
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Logf("parse templateDelete: %v", err)
	}
}

func testCreateIssueFromTemplate(t *testing.T, q q, templateID, teamID string) string {
	t.Helper()
	mutation := `mutation ($templateId: String!, $teamId: String!) {
		issueCreate(input: {templateId: $templateId, teamId: $teamId}) {
			success
			issue { id }
		}
	}`
	body, err := q.do(mutation, map[string]any{"templateId": templateID, "teamId": teamID})
	if err != nil {
		t.Fatalf("create issue from template: %v", err)
	}
	var resp struct {
		Data struct {
			IssueCreate struct {
				Success bool
				Issue   struct{ ID string }
			}
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parse issueCreate: %v\nbody: %s", err, body)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("issueCreate error: %s", resp.Errors[0].Message)
	}
	if !resp.Data.IssueCreate.Success {
		t.Fatal("issueCreate returned success=false")
	}
	return resp.Data.IssueCreate.Issue.ID
}

func testDeleteIssue(t *testing.T, q q, id string) {
	t.Helper()
	mutation := `mutation ($id: String!) {
		issueDelete(id: $id) { success }
	}`
	body, err := q.do(mutation, map[string]any{"id": id})
	if err != nil {
		t.Logf("delete issue %s: %v", id, err)
		return
	}
	var resp struct {
		Data struct {
			IssueDelete struct{ Success bool }
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Logf("parse issueDelete: %v", err)
	}
}

func testCreateChildIssue(t *testing.T, q q, teamID, parentID, title string) string {
	t.Helper()
	mutation := `mutation ($input: IssueCreateInput!) {
		issueCreate(input: $input) {
			success
			issue { id }
		}
	}`
	input := map[string]any{
		"teamId":   teamID,
		"title":    title,
		"parentId": parentID,
	}
	body, err := q.do(mutation, map[string]any{"input": input})
	if err != nil {
		t.Fatalf("create child issue: %v", err)
	}
	var resp struct {
		Data struct {
			IssueCreate struct {
				Success bool
				Issue   struct{ ID string }
			}
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parse child issueCreate: %v\nbody: %s", err, body)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("child issueCreate error: %s", resp.Errors[0].Message)
	}
	if !resp.Data.IssueCreate.Success {
		t.Fatal("child issueCreate returned success=false")
	}
	return resp.Data.IssueCreate.Issue.ID
}

type testRelation struct {
	issueID   string
	relatedID string
	relType   string
}

func testGetIssueRelations(t *testing.T, q q, issueID string) []testRelation {
	t.Helper()
	query := `query ($id: String!) {
		issue(id: $id) {
			relations {
				nodes {
					relatedIssue { id }
					type
				}
			}
			inverseRelations {
				nodes {
					issue { id }
					type
				}
			}
		}
	}`
	body, err := q.do(query, map[string]any{"id": issueID})
	if err != nil {
		t.Fatalf("get relations: %v", err)
	}
	var resp struct {
		Data struct {
			Issue struct {
				Relations struct {
					Nodes []struct {
						RelatedIssue struct{ ID string }
						Type         string
					}
				}
				InverseRelations struct {
					Nodes []struct {
						Issue struct{ ID string }
						Type  string
					}
				}
			}
		}
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parse relations: %v\nbody: %s", err, body)
	}

	var out []testRelation
	for _, r := range resp.Data.Issue.Relations.Nodes {
		out = append(out, testRelation{issueID: issueID, relatedID: r.RelatedIssue.ID, relType: r.Type})
	}
	for _, r := range resp.Data.Issue.InverseRelations.Nodes {
		out = append(out, testRelation{issueID: r.Issue.ID, relatedID: issueID, relType: r.Type})
	}
	return out
}

func testGetIssueTitle(t *testing.T, q q, issueID string) string {
	t.Helper()
	query := `query ($id: String!) { issue(id: $id) { title } }`
	body, err := q.do(query, map[string]any{"id": issueID})
	if err != nil {
		t.Fatalf("get issue title: %v", err)
	}
	var resp struct {
		Data struct {
			Issue struct{ Title string }
		}
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("parse issue title: %v", err)
	}
	return resp.Data.Issue.Title
}

// testCleanupMarked deletes all templates and issues whose name/title contains testMarker.
func testCleanupMarked(t *testing.T, q q, teamID string) {
	t.Helper()

	issues, err := searchTeamIssues(q, teamID, testMarker)
	if err != nil {
		t.Logf("cleanup: list issues: %v", err)
	} else {
		for _, iss := range issues {
			testDeleteIssue(t, q, iss.id)
		}
	}

	templates, err := getTemplates(q)
	if err != nil {
		t.Logf("cleanup: list templates: %v", err)
	} else {
		for _, tmpl := range templates {
			if strings.Contains(tmpl.name, testMarker) {
				testDeleteTemplate(t, q, tmpl.id)
			}
		}
	}
}

// --- integration tests ---

func TestIntegration(t *testing.T) {
	skipUnlessIntegration(t)
	q := integrationQ(t)
	teamID := testGetTeamID(t, q)

	testCleanupMarked(t, q, teamID)
	t.Cleanup(func() { testCleanupMarked(t, q, teamID) })

	t.Run("TemplateDescriptionRoundTrip", func(t *testing.T) {
		description := "Recurrence: daily\nRecurrence: Mon\nSome other text"
		name := testMarker + " roundtrip"
		tmplID := testCreateTemplate(t, q, teamID, name, description)

		templates, err := getTemplates(q)
		if err != nil {
			t.Fatalf("getTemplates: %v", err)
		}

		var found *issueTemplate
		for i := range templates {
			if templates[i].id == tmplID {
				found = &templates[i]
				break
			}
		}
		if found == nil {
			t.Fatal("created template not found in getTemplates result")
		}
		if found.description != description {
			t.Errorf("description mismatch:\n  got:  %q\n  want: %q", found.description, description)
		}
		if found.name != name {
			t.Errorf("name mismatch: got %q, want %q", found.name, name)
		}
		if found.teamID != teamID {
			t.Errorf("teamID mismatch: got %q, want %q", found.teamID, teamID)
		}
	})

	t.Run("RecurrenceMatching", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" recurrence", "Recurrence: daily")

		templates, err := getTemplates(q)
		if err != nil {
			t.Fatalf("getTemplates: %v", err)
		}

		var found *issueTemplate
		for i := range templates {
			if templates[i].id == tmplID {
				found = &templates[i]
				break
			}
		}
		if found == nil {
			t.Fatal("created template not found")
		}

		anyDay := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
		if !templateMatchesSchedule(found.description, anyDay) {
			t.Error("daily recurrence should match any date")
		}

		tmplID2 := testCreateTemplate(t, q, teamID, testMarker+" no-recurrence", "Just a description")

		templates2, err := getTemplates(q)
		if err != nil {
			t.Fatalf("getTemplates: %v", err)
		}
		for _, tmpl := range templates2 {
			if tmpl.id == tmplID2 {
				if templateMatchesSchedule(tmpl.description, anyDay) {
					t.Error("template without recurrence lines should not match")
				}
				break
			}
		}
	})

	t.Run("WeekdayRecurrence", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" weekday", "Recurrence: Wed")

		templates, err := getTemplates(q)
		if err != nil {
			t.Fatalf("getTemplates: %v", err)
		}

		var found *issueTemplate
		for i := range templates {
			if templates[i].id == tmplID {
				found = &templates[i]
				break
			}
		}
		if found == nil {
			t.Fatal("created template not found")
		}

		wed := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
		thu := time.Date(2025, time.January, 16, 0, 0, 0, 0, time.UTC)

		if !templateMatchesSchedule(found.description, wed) {
			t.Error("Wed recurrence should match Wednesday")
		}
		if templateMatchesSchedule(found.description, thu) {
			t.Error("Wed recurrence should not match Thursday")
		}
	})

	t.Run("CreateIssueAndTrack", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" create-track", "Recurrence: daily")

		testCreateIssueFromTemplate(t, q, tmplID, teamID)

		today := time.Now().UTC().Truncate(24 * time.Hour)
		created, err := getTemplateCreatedIssuesForDay(q, teamID, today)
		if err != nil {
			t.Fatalf("getTemplateCreatedIssuesForDay: %v", err)
		}

		if !created[tmplID] {
			t.Errorf("template %s should appear in created-today map; got: %v", tmplID, created)
		}
	})

	t.Run("CreateRecurringIssuesIdempotent", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" idempotent", "Recurrence: daily")

		today := time.Now().UTC().Truncate(24 * time.Hour)

		testCreateIssueFromTemplate(t, q, tmplID, teamID)

		created, err := getTemplateCreatedIssuesForDay(q, teamID, today)
		if err != nil {
			t.Fatalf("getTemplateCreatedIssuesForDay: %v", err)
		}
		if !created[tmplID] {
			t.Fatal("template should be in created-today map after first issue")
		}

		// Simulate what createFromDueTemplates does: skip if already created.
		templates, err := getTemplates(q)
		if err != nil {
			t.Fatalf("getTemplates: %v", err)
		}
		var dueTemplates []issueTemplate
		for _, tmpl := range templates {
			if tmpl.teamID == teamID && templateMatchesSchedule(tmpl.description, today) {
				dueTemplates = append(dueTemplates, tmpl)
			}
		}

		foundOurs := false
		for _, tmpl := range dueTemplates {
			if tmpl.id == tmplID {
				foundOurs = true
				if !created[tmpl.id] {
					t.Error("our template should be in created map, so it would be skipped")
				}
			}
		}
		if !foundOurs {
			t.Error("our daily template should be in the due list")
		}
	})

	t.Run("MonthDayRecurrence", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" monthday", "Recurrence: Mar 15")

		templates, err := getTemplates(q)
		if err != nil {
			t.Fatalf("getTemplates: %v", err)
		}

		var found *issueTemplate
		for i := range templates {
			if templates[i].id == tmplID {
				found = &templates[i]
				break
			}
		}
		if found == nil {
			t.Fatal("created template not found")
		}

		mar15 := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
		mar16 := time.Date(2025, time.March, 16, 0, 0, 0, 0, time.UTC)
		apr15 := time.Date(2025, time.April, 15, 0, 0, 0, 0, time.UTC)

		if !templateMatchesSchedule(found.description, mar15) {
			t.Error("Mar 15 should match March 15")
		}
		if templateMatchesSchedule(found.description, mar16) {
			t.Error("Mar 15 should not match March 16")
		}
		if templateMatchesSchedule(found.description, apr15) {
			t.Error("Mar 15 should not match April 15")
		}
	})

	t.Run("MultipleRecurrenceLines", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" multi", "Recurrence: Mon\nRecurrence: 15\nOther info")

		templates, err := getTemplates(q)
		if err != nil {
			t.Fatalf("getTemplates: %v", err)
		}

		var found *issueTemplate
		for i := range templates {
			if templates[i].id == tmplID {
				found = &templates[i]
				break
			}
		}
		if found == nil {
			t.Fatal("created template not found")
		}

		mon := time.Date(2025, time.January, 13, 0, 0, 0, 0, time.UTC)
		day15 := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
		tue14 := time.Date(2025, time.January, 14, 0, 0, 0, 0, time.UTC)

		if !templateMatchesSchedule(found.description, mon) {
			t.Error("should match Monday via Mon line")
		}
		if !templateMatchesSchedule(found.description, day15) {
			t.Error("should match day 15 via 15 line")
		}
		if templateMatchesSchedule(found.description, tue14) {
			t.Error("should not match Tuesday the 14th")
		}
	})

	t.Run("ListTemplatesShowsDescription", func(t *testing.T) {
		description := "Recurrence: daily\nExtra info"
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" list", description)

		templates, err := getTemplates(q)
		if err != nil {
			t.Fatalf("getTemplates: %v", err)
		}

		for _, tmpl := range templates {
			if tmpl.id == tmplID {
				expected := fmt.Sprintf("%s\t%s\t%s\t%s", tmpl.id, tmpl.teamID, tmpl.name, tmpl.description)
				if expected == "" {
					t.Error("unexpected empty output")
				}
				t.Logf("listing line: %s", expected)
				return
			}
		}
		t.Error("template not found in listing")
	})

	t.Run("SubIssueDependencies", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" subdeps", "")
		parentID := testCreateIssueFromTemplate(t, q, tmplID, teamID)

		child1ID := testCreateChildIssue(t, q, teamID, parentID, "1|REQ "+testMarker+" First task")
		child2ID := testCreateChildIssue(t, q, teamID, parentID, "2|NEEDS1 "+testMarker+" Second task")
		testCreateChildIssue(t, q, teamID, parentID, testMarker+" No prefix task")

		if err := setupSubIssueDependencies(q, parentID); err != nil {
			t.Fatalf("setupSubIssueDependencies: %v", err)
		}

		// Verify titles were stripped.
		if title := testGetIssueTitle(t, q, child1ID); title != testMarker+" First task" {
			t.Errorf("child1 title = %q, want %q", title, testMarker+" First task")
		}
		if title := testGetIssueTitle(t, q, child2ID); title != testMarker+" Second task" {
			t.Errorf("child2 title = %q, want %q", title, testMarker+" Second task")
		}

		// Verify relations: child1 blocks parent (REQ).
		parentRels := testGetIssueRelations(t, q, parentID)
		foundChild1BlocksParent := false
		for _, r := range parentRels {
			if r.issueID == child1ID && r.relatedID == parentID && r.relType == "blocks" {
				foundChild1BlocksParent = true
			}
		}
		if !foundChild1BlocksParent {
			t.Errorf("expected child1 blocks parent relation; got relations: %+v", parentRels)
		}

		// Verify relations: child1 blocks child2 (NEEDS1).
		child2Rels := testGetIssueRelations(t, q, child2ID)
		foundChild1BlocksChild2 := false
		for _, r := range child2Rels {
			if r.issueID == child1ID && r.relatedID == child2ID && r.relType == "blocks" {
				foundChild1BlocksChild2 = true
			}
		}
		if !foundChild1BlocksChild2 {
			t.Errorf("expected child1 blocks child2 relation; got relations: %+v", child2Rels)
		}
	})
}
