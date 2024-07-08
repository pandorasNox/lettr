package github

import (
	"context"

	"github.com/google/go-github/v62/github"
)

type IssueRequest github.IssueRequest

func CreateIssue(ctx context.Context, githubToken string, ir IssueRequest) error {
	client := github.NewTokenClient(ctx, githubToken)

	//issueTitle := "new word suggestion test"
	//ir := github.IssueRequest{Title: &issueTitle}

	gir := github.IssueRequest(ir)
	_, _, err := client.Issues.Create(ctx, "pandorasNox", "lettr", &gir)

	// fmt.Printf("Issue: %v\n", issue)
	// fmt.Printf("Response: %v\n", res)
	// fmt.Printf("Error: %v\n", err)

	return err
}
