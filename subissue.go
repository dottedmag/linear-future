package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type subIssueProblem struct {
	title   string
	problem string
}

// validateSubIssuePrefixes checks sub-issue titles for structural problems:
// duplicate IDs and dangling NEEDS references.
func validateSubIssuePrefixes(titles []string) []subIssueProblem {
	var problems []subIssueProblem
	ids := map[int]bool{}

	type parsed struct {
		title  string
		prefix subIssuePrefix
	}
	var items []parsed
	for _, title := range titles {
		p := parseSubIssuePrefix(title)
		items = append(items, parsed{title: title, prefix: p})
		if !p.hasPrefix {
			continue
		}
		if ids[p.id] {
			problems = append(problems, subIssueProblem{title: title, problem: fmt.Sprintf("duplicate ID %d", p.id)})
		}
		ids[p.id] = true
	}
	for _, item := range items {
		if !item.prefix.hasPrefix {
			continue
		}
		for _, need := range item.prefix.needs {
			if !ids[need] {
				problems = append(problems, subIssueProblem{
					title:   item.title,
					problem: fmt.Sprintf("sub-issue %d NEEDS %d, but no sub-issue with that ID", item.prefix.id, need),
				})
			}
		}
	}
	return problems
}

// subIssuePrefix represents the parsed prefix from a sub-issue title.
type subIssuePrefix struct {
	id       int    // numeric ID of this sub-issue (0 if no prefix)
	req      bool   // parent depends on this sub-issue
	needs    []int  // IDs of sub-issues this one depends on
	title    string // title with prefix stripped
	hasPrefix bool  // whether the title had a prefix at all
}

// prefixRx matches the sub-issue prefix: 123(|REQ)?(|NEEDS456)* followed by a space and the rest of the title.
var prefixRx = regexp.MustCompile(`^(\d+)((?:\|REQ)?(?:\|NEEDS\d+)*)\s+(.+)$`)

var needsRx = regexp.MustCompile(`\|NEEDS(\d+)`)

func parseSubIssuePrefix(title string) subIssuePrefix {
	m := prefixRx.FindStringSubmatch(title)
	if m == nil {
		return subIssuePrefix{title: title}
	}

	id, _ := strconv.Atoi(m[1])
	flags := m[2]
	rest := m[3]

	p := subIssuePrefix{
		id:        id,
		req:       strings.Contains(flags, "|REQ"),
		title:     rest,
		hasPrefix: true,
	}

	for _, nm := range needsRx.FindAllStringSubmatch(flags, -1) {
		n, _ := strconv.Atoi(nm[1])
		p.needs = append(p.needs, n)
	}

	return p
}

// setupSubIssueDependencies parses sub-issue title prefixes, creates dependency
// relations, and strips prefixes from titles. Call after creating an issue from
// a template.
func setupSubIssueDependencies(q q, parentID string) error {
	children, err := getChildIssues(q, parentID)
	if err != nil {
		return fmt.Errorf("fetching sub-issues: %w", err)
	}
	if len(children) == 0 {
		return nil
	}

	// Parse all prefixes and build a map from numeric ID to Linear issue ID.
	type parsed struct {
		sub    subIssue
		prefix subIssuePrefix
	}
	var items []parsed
	idMap := map[int]string{} // numeric prefix ID -> Linear issue ID

	for _, child := range children {
		p := parseSubIssuePrefix(child.title)
		items = append(items, parsed{sub: child, prefix: p})
		if p.hasPrefix && p.id > 0 {
			idMap[p.id] = child.id
		}
	}

	// Create relations and strip prefixes.
	for _, item := range items {
		if !item.prefix.hasPrefix {
			continue
		}

		// REQ: parent depends on this sub-issue (this sub-issue blocks parent).
		if item.prefix.req {
			fmt.Printf("  sub-issue %d blocks parent\n", item.prefix.id)
			if err := createBlocksRelation(q, item.sub.id, parentID); err != nil {
				return fmt.Errorf("creating REQ relation for sub-issue %d: %w", item.prefix.id, err)
			}
		}

		// NEEDS: this sub-issue depends on another (the other blocks this one).
		for _, needID := range item.prefix.needs {
			blockerLinearID, ok := idMap[needID]
			if !ok {
				return fmt.Errorf("sub-issue %d NEEDS %d, but no sub-issue with that ID found", item.prefix.id, needID)
			}
			fmt.Printf("  sub-issue %d depends on sub-issue %d\n", item.prefix.id, needID)
			if err := createBlocksRelation(q, blockerLinearID, item.sub.id); err != nil {
				return fmt.Errorf("creating NEEDS relation for sub-issue %d -> %d: %w", item.prefix.id, needID, err)
			}
		}

		// Strip the prefix from the title.
		if item.prefix.title != item.sub.title {
			fmt.Printf("  renaming sub-issue %q -> %q\n", item.sub.title, item.prefix.title)
			if err := updateTitle(q, item.sub.id, item.prefix.title); err != nil {
				return fmt.Errorf("stripping prefix from sub-issue %d: %w", item.prefix.id, err)
			}
		}
	}

	return nil
}
