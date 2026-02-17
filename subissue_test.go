package main

import (
	"reflect"
	"testing"
)

func TestParseSubIssuePrefix_NoPrefix(t *testing.T) {
	p := parseSubIssuePrefix("Do the thing")
	if p.hasPrefix {
		t.Error("should not have prefix")
	}
	if p.title != "Do the thing" {
		t.Errorf("title = %q, want %q", p.title, "Do the thing")
	}
}

func TestParseSubIssuePrefix_IDOnly(t *testing.T) {
	p := parseSubIssuePrefix("1 Do the thing")
	if !p.hasPrefix {
		t.Error("should have prefix")
	}
	if p.id != 1 {
		t.Errorf("id = %d, want 1", p.id)
	}
	if p.req {
		t.Error("should not be req")
	}
	if len(p.needs) != 0 {
		t.Errorf("needs = %v, want empty", p.needs)
	}
	if p.title != "Do the thing" {
		t.Errorf("title = %q, want %q", p.title, "Do the thing")
	}
}

func TestParseSubIssuePrefix_Req(t *testing.T) {
	p := parseSubIssuePrefix("2|REQ Do the blocking thing")
	if !p.hasPrefix {
		t.Error("should have prefix")
	}
	if p.id != 2 {
		t.Errorf("id = %d, want 2", p.id)
	}
	if !p.req {
		t.Error("should be req")
	}
	if len(p.needs) != 0 {
		t.Errorf("needs = %v, want empty", p.needs)
	}
	if p.title != "Do the blocking thing" {
		t.Errorf("title = %q, want %q", p.title, "Do the blocking thing")
	}
}

func TestParseSubIssuePrefix_Needs(t *testing.T) {
	p := parseSubIssuePrefix("3|NEEDS1|NEEDS2 Do the last thing")
	if !p.hasPrefix {
		t.Error("should have prefix")
	}
	if p.id != 3 {
		t.Errorf("id = %d, want 3", p.id)
	}
	if p.req {
		t.Error("should not be req")
	}
	if !reflect.DeepEqual(p.needs, []int{1, 2}) {
		t.Errorf("needs = %v, want [1 2]", p.needs)
	}
	if p.title != "Do the last thing" {
		t.Errorf("title = %q, want %q", p.title, "Do the last thing")
	}
}

func TestParseSubIssuePrefix_ReqAndNeeds(t *testing.T) {
	p := parseSubIssuePrefix("4|REQ|NEEDS1 Critical path")
	if !p.hasPrefix {
		t.Error("should have prefix")
	}
	if p.id != 4 {
		t.Errorf("id = %d, want 4", p.id)
	}
	if !p.req {
		t.Error("should be req")
	}
	if !reflect.DeepEqual(p.needs, []int{1}) {
		t.Errorf("needs = %v, want [1]", p.needs)
	}
	if p.title != "Critical path" {
		t.Errorf("title = %q, want %q", p.title, "Critical path")
	}
}

func TestParseSubIssuePrefix_MultiDigitIDs(t *testing.T) {
	p := parseSubIssuePrefix("12|NEEDS345|NEEDS67 Complex task")
	if !p.hasPrefix {
		t.Error("should have prefix")
	}
	if p.id != 12 {
		t.Errorf("id = %d, want 12", p.id)
	}
	if !reflect.DeepEqual(p.needs, []int{345, 67}) {
		t.Errorf("needs = %v, want [345 67]", p.needs)
	}
	if p.title != "Complex task" {
		t.Errorf("title = %q, want %q", p.title, "Complex task")
	}
}

func TestParseSubIssuePrefix_TitleStartsWithNumber(t *testing.T) {
	// A title like "42 is the answer" — this has a prefix with ID 42
	p := parseSubIssuePrefix("42 is the answer")
	if !p.hasPrefix {
		t.Error("should have prefix")
	}
	if p.id != 42 {
		t.Errorf("id = %d, want 42", p.id)
	}
	if p.title != "is the answer" {
		t.Errorf("title = %q, want %q", p.title, "is the answer")
	}
}

func TestParseSubIssuePrefix_EmptyTitle(t *testing.T) {
	p := parseSubIssuePrefix("")
	if p.hasPrefix {
		t.Error("should not have prefix")
	}
	if p.title != "" {
		t.Errorf("title = %q, want empty", p.title)
	}
}

func TestParseSubIssuePrefix_NumberAtEnd(t *testing.T) {
	// Just a number with no space and title after — no prefix
	p := parseSubIssuePrefix("42")
	if p.hasPrefix {
		t.Error("should not have prefix — no title after number")
	}
	if p.title != "42" {
		t.Errorf("title = %q, want %q", p.title, "42")
	}
}
