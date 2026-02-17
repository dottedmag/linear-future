package main

import (
	"fmt"
	"os"
)

func runListTemplates(token string) int {
	q := q{token}
	templates, err := getTemplates(q)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list templates: %v\n", err)
		return 1
	}
	for _, t := range templates {
		fmt.Printf("%s\t%s\t%s\t%s\n", t.id, t.teamID, t.name, t.description)
	}
	return 0
}
