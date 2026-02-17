package main

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestParseSubIssuePrefix_NoPrefix(t *testing.T) {
	p := parseSubIssuePrefix("Do the thing")
	assert.False(t, p.hasPrefix)
	assert.Equal(t, "Do the thing", p.title)
}

func TestParseSubIssuePrefix_IDOnly(t *testing.T) {
	p := parseSubIssuePrefix("1 Do the thing")
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 1, p.id)
	assert.False(t, p.req)
	assert.Equal(t, 0, len(p.needs))
	assert.Equal(t, "Do the thing", p.title)
}

func TestParseSubIssuePrefix_Req(t *testing.T) {
	p := parseSubIssuePrefix("2|REQ Do the blocking thing")
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 2, p.id)
	assert.True(t, p.req)
	assert.Equal(t, 0, len(p.needs))
	assert.Equal(t, "Do the blocking thing", p.title)
}

func TestParseSubIssuePrefix_Needs(t *testing.T) {
	p := parseSubIssuePrefix("3|NEEDS1|NEEDS2 Do the last thing")
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 3, p.id)
	assert.False(t, p.req)
	assert.Equal(t, []int{1, 2}, p.needs)
	assert.Equal(t, "Do the last thing", p.title)
}

func TestParseSubIssuePrefix_ReqAndNeeds(t *testing.T) {
	p := parseSubIssuePrefix("4|REQ|NEEDS1 Critical path")
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 4, p.id)
	assert.True(t, p.req)
	assert.Equal(t, []int{1}, p.needs)
	assert.Equal(t, "Critical path", p.title)
}

func TestParseSubIssuePrefix_MultiDigitIDs(t *testing.T) {
	p := parseSubIssuePrefix("12|NEEDS345|NEEDS67 Complex task")
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 12, p.id)
	assert.Equal(t, []int{345, 67}, p.needs)
	assert.Equal(t, "Complex task", p.title)
}

func TestParseSubIssuePrefix_TitleStartsWithNumber(t *testing.T) {
	// A title like "42 is the answer" — this has a prefix with ID 42
	p := parseSubIssuePrefix("42 is the answer")
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 42, p.id)
	assert.Equal(t, "is the answer", p.title)
}

func TestParseSubIssuePrefix_EmptyTitle(t *testing.T) {
	p := parseSubIssuePrefix("")
	assert.False(t, p.hasPrefix)
	assert.Equal(t, "", p.title)
}

func TestParseSubIssuePrefix_NumberAtEnd(t *testing.T) {
	// Just a number with no space and title after — no prefix
	p := parseSubIssuePrefix("42")
	assert.False(t, p.hasPrefix)
	assert.Equal(t, "42", p.title)
}
