package main

import (
	"fmt"
	"os"
)

func realMain() int {
	token := os.Getenv("LINEAR_API_KEY")
	if len(os.Args) == 2 && os.Args[1] == "--list-templates" {
		return runListTemplates(token)
	}
	if token == "" || len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: LINEAR_API_KEY=lin_api_... linear-future <team name>\n")
		return 2
	}
	retCode := 0
	for _, teamName := range os.Args[1:] {
		if err := createScheduledTeamIssues(token, teamName); err != nil {
			fmt.Fprintln(os.Stderr, err)
			retCode = 1
		}
	}
	return retCode
}

func main() {
	os.Exit(realMain())
}
