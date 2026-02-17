package main

import (
	"flag"
	"fmt"
	"os"
)

func realMain() int {
	listTemplates := flag.Bool("list-templates", false, "List all templates")
	list := flag.Bool("list", false, "Show template schedules, trigger dates, and sub-issue validation")
	flag.Parse()

	token := os.Getenv("LINEAR_API_KEY")

	if *listTemplates {
		return runListTemplates(token)
	}
	if *list {
		return runList(token)
	}

	if token == "" || flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: LINEAR_API_KEY=lin_api_... linear-future [flags] <team name>...\n")
		flag.PrintDefaults()
		return 2
	}
	retCode := 0
	for _, teamName := range flag.Args() {
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
