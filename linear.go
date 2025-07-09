package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
