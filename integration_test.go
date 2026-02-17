package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/assert/v2"
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
	assert.NotEqual(t, "", token, "LINEAR_API_KEY must be set for integration tests")
	return q{token}
}

// --- test helpers ---

func testGetTeamID(t *testing.T, q q) string {
	t.Helper()
	body, err := q.do(`query { teams { nodes { id } } }`, nil)
	assert.NoError(t, err)
	var resp struct {
		Data struct {
			Teams struct {
				Nodes []struct{ ID string }
			}
		}
	}
	assert.NoError(t, json.Unmarshal(body, &resp))
	assert.True(t, len(resp.Data.Teams.Nodes) > 0, "no teams in workspace")
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
	assert.NoError(t, err)
	var resp struct {
		Data struct {
			TemplateCreate struct {
				Success  bool
				Template struct{ ID string }
			}
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	assert.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, 0, len(resp.Errors))
	assert.True(t, resp.Data.TemplateCreate.Success)
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
	assert.NoError(t, err)
	var resp struct {
		Data struct {
			IssueCreate struct {
				Success bool
				Issue   struct{ ID string }
			}
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	assert.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, 0, len(resp.Errors))
	assert.True(t, resp.Data.IssueCreate.Success)
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
	assert.NoError(t, err)
	var resp struct {
		Data struct {
			IssueCreate struct {
				Success bool
				Issue   struct{ ID string }
			}
		} `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	assert.NoError(t, json.Unmarshal(body, &resp))
	assert.Equal(t, 0, len(resp.Errors))
	assert.True(t, resp.Data.IssueCreate.Success)
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
	assert.NoError(t, err)
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
	assert.NoError(t, json.Unmarshal(body, &resp))

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
	assert.NoError(t, err)
	var resp struct {
		Data struct {
			Issue struct{ Title string }
		}
	}
	assert.NoError(t, json.Unmarshal(body, &resp))
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

func findTemplate(t *testing.T, templates []issueTemplate, id string) issueTemplate {
	t.Helper()
	for _, tmpl := range templates {
		if tmpl.id == id {
			return tmpl
		}
	}
	t.Fatalf("template %s not found", id)
	return issueTemplate{}
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
		assert.NoError(t, err)

		found := findTemplate(t, templates, tmplID)
		assert.Equal(t, description, found.description)
		assert.Equal(t, name, found.name)
		assert.Equal(t, teamID, found.teamID)
	})

	t.Run("RecurrenceMatching", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" recurrence", "Recurrence: daily")

		templates, err := getTemplates(q)
		assert.NoError(t, err)

		found := findTemplate(t, templates, tmplID)

		anyDay := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
		assert.True(t, templateMatchesSchedule(found.description, anyDay))

		tmplID2 := testCreateTemplate(t, q, teamID, testMarker+" no-recurrence", "Just a description")

		templates2, err := getTemplates(q)
		assert.NoError(t, err)

		found2 := findTemplate(t, templates2, tmplID2)
		assert.False(t, templateMatchesSchedule(found2.description, anyDay))
	})

	t.Run("WeekdayRecurrence", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" weekday", "Recurrence: Wed")

		templates, err := getTemplates(q)
		assert.NoError(t, err)

		found := findTemplate(t, templates, tmplID)

		wed := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
		thu := time.Date(2025, time.January, 16, 0, 0, 0, 0, time.UTC)

		assert.True(t, templateMatchesSchedule(found.description, wed))
		assert.False(t, templateMatchesSchedule(found.description, thu))
	})

	t.Run("CreateIssueAndTrack", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" create-track", "Recurrence: daily")

		testCreateIssueFromTemplate(t, q, tmplID, teamID)

		today := time.Now().UTC().Truncate(24 * time.Hour)
		created, err := getTemplateCreatedIssuesForDay(q, teamID, today)
		assert.NoError(t, err)
		assert.True(t, created[tmplID])
	})

	t.Run("CreateRecurringIssuesIdempotent", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" idempotent", "Recurrence: daily")

		today := time.Now().UTC().Truncate(24 * time.Hour)

		testCreateIssueFromTemplate(t, q, tmplID, teamID)

		created, err := getTemplateCreatedIssuesForDay(q, teamID, today)
		assert.NoError(t, err)
		assert.True(t, created[tmplID])

		// Simulate what createFromDueTemplates does: skip if already created.
		templates, err := getTemplates(q)
		assert.NoError(t, err)
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
				assert.True(t, created[tmpl.id])
			}
		}
		assert.True(t, foundOurs, "our daily template should be in the due list")
	})

	t.Run("MonthDayRecurrence", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" monthday", "Recurrence: Mar 15")

		templates, err := getTemplates(q)
		assert.NoError(t, err)

		found := findTemplate(t, templates, tmplID)

		mar15 := time.Date(2025, time.March, 15, 0, 0, 0, 0, time.UTC)
		mar16 := time.Date(2025, time.March, 16, 0, 0, 0, 0, time.UTC)
		apr15 := time.Date(2025, time.April, 15, 0, 0, 0, 0, time.UTC)

		assert.True(t, templateMatchesSchedule(found.description, mar15))
		assert.False(t, templateMatchesSchedule(found.description, mar16))
		assert.False(t, templateMatchesSchedule(found.description, apr15))
	})

	t.Run("MultipleRecurrenceLines", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" multi", "Recurrence: Mon\nRecurrence: 15\nOther info")

		templates, err := getTemplates(q)
		assert.NoError(t, err)

		found := findTemplate(t, templates, tmplID)

		mon := time.Date(2025, time.January, 13, 0, 0, 0, 0, time.UTC)
		day15 := time.Date(2025, time.January, 15, 0, 0, 0, 0, time.UTC)
		tue14 := time.Date(2025, time.January, 14, 0, 0, 0, 0, time.UTC)

		assert.True(t, templateMatchesSchedule(found.description, mon))
		assert.True(t, templateMatchesSchedule(found.description, day15))
		assert.False(t, templateMatchesSchedule(found.description, tue14))
	})

	t.Run("ListTemplatesShowsDescription", func(t *testing.T) {
		description := "Recurrence: daily\nExtra info"
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" list", description)

		templates, err := getTemplates(q)
		assert.NoError(t, err)

		found := findTemplate(t, templates, tmplID)
		expected := fmt.Sprintf("%s\t%s\t%s\t%s", found.id, found.teamID, found.name, found.description)
		assert.NotEqual(t, "", expected)
	})

	t.Run("SubIssueDependencies", func(t *testing.T) {
		tmplID := testCreateTemplate(t, q, teamID, testMarker+" subdeps", "")
		parentID := testCreateIssueFromTemplate(t, q, tmplID, teamID)

		child1ID := testCreateChildIssue(t, q, teamID, parentID, "1|REQ "+testMarker+" First task")
		child2ID := testCreateChildIssue(t, q, teamID, parentID, "2|NEEDS1 "+testMarker+" Second task")
		testCreateChildIssue(t, q, teamID, parentID, testMarker+" No prefix task")

		assert.NoError(t, setupSubIssueDependencies(q, parentID))

		// Verify titles were stripped.
		assert.Equal(t, testMarker+" First task", testGetIssueTitle(t, q, child1ID))
		assert.Equal(t, testMarker+" Second task", testGetIssueTitle(t, q, child2ID))

		// Verify relations: child1 blocks parent (REQ).
		parentRels := testGetIssueRelations(t, q, parentID)
		foundChild1BlocksParent := false
		for _, r := range parentRels {
			if r.issueID == child1ID && r.relatedID == parentID && r.relType == "blocks" {
				foundChild1BlocksParent = true
			}
		}
		assert.True(t, foundChild1BlocksParent, "expected child1 blocks parent relation")

		// Verify relations: child1 blocks child2 (NEEDS1).
		child2Rels := testGetIssueRelations(t, q, child2ID)
		foundChild1BlocksChild2 := false
		for _, r := range child2Rels {
			if r.issueID == child1ID && r.relatedID == child2ID && r.relType == "blocks" {
				foundChild1BlocksChild2 = true
			}
		}
		assert.True(t, foundChild1BlocksChild2, "expected child1 blocks child2 relation")
	})
}
