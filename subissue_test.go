package main

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestParseSubIssuePrefix_NoPrefix(t *testing.T) {
	p, err := parseSubIssuePrefix("Do the thing")
	assert.NoError(t, err)
	assert.False(t, p.hasPrefix)
	assert.Equal(t, "Do the thing", p.title)
}

func TestParseSubIssuePrefix_IDOnly(t *testing.T) {
	p, err := parseSubIssuePrefix("1 Do the thing")
	assert.NoError(t, err)
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 1, p.id)
	assert.False(t, p.req)
	assert.Equal(t, 0, len(p.needs))
	assert.Equal(t, "Do the thing", p.title)
}

func TestParseSubIssuePrefix_Req(t *testing.T) {
	p, err := parseSubIssuePrefix("2|REQ Do the blocking thing")
	assert.NoError(t, err)
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 2, p.id)
	assert.True(t, p.req)
	assert.Equal(t, 0, len(p.needs))
	assert.Equal(t, "Do the blocking thing", p.title)
}

func TestParseSubIssuePrefix_Deps(t *testing.T) {
	p, err := parseSubIssuePrefix("3|DEPS1|DEPS2 Do the last thing")
	assert.NoError(t, err)
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 3, p.id)
	assert.False(t, p.req)
	assert.Equal(t, []int{1, 2}, p.needs)
	assert.Equal(t, "Do the last thing", p.title)
}

func TestParseSubIssuePrefix_ReqAndDeps(t *testing.T) {
	p, err := parseSubIssuePrefix("4|REQ|DEPS1 Critical path")
	assert.NoError(t, err)
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 4, p.id)
	assert.True(t, p.req)
	assert.Equal(t, []int{1}, p.needs)
	assert.Equal(t, "Critical path", p.title)
}

func TestParseSubIssuePrefix_MultiDigitIDs(t *testing.T) {
	p, err := parseSubIssuePrefix("12|DEPS345|DEPS67 Complex task")
	assert.NoError(t, err)
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 12, p.id)
	assert.Equal(t, []int{345, 67}, p.needs)
	assert.Equal(t, "Complex task", p.title)
}

func TestParseSubIssuePrefix_TitleStartsWithNumber(t *testing.T) {
	// A title like "42 is the answer" — this has a prefix with ID 42
	p, err := parseSubIssuePrefix("42 is the answer")
	assert.NoError(t, err)
	assert.True(t, p.hasPrefix)
	assert.Equal(t, 42, p.id)
	assert.Equal(t, "is the answer", p.title)
}

func TestParseSubIssuePrefix_EmptyTitle(t *testing.T) {
	p, err := parseSubIssuePrefix("")
	assert.NoError(t, err)
	assert.False(t, p.hasPrefix)
	assert.Equal(t, "", p.title)
}

func TestParseSubIssuePrefix_NumberAtEnd(t *testing.T) {
	// Just a number with no space and title after — invalid prefix
	_, err := parseSubIssuePrefix("42")
	assert.Error(t, err)
}

func TestParseSubIssuePrefix_InvalidPrefix(t *testing.T) {
	// Starts with a digit but has invalid flags
	_, err := parseSubIssuePrefix("3|BOGUS Do something")
	assert.Error(t, err)
}

func TestParseSubIssuePrefix_SelfDependency(t *testing.T) {
	_, err := parseSubIssuePrefix("6|DEPS6 Do something")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "depends on itself")
}
