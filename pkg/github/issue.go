package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v72/github"
)

func CreateWordSuggestionIssue(ctx context.Context, githubToken string, word, language, action, message string) error {
	err := validateSuggestionInputs(word, language, action, message) //TODO: maybe move or overload createSuggestionBody ???
	if err != nil {
		return fmt.Errorf("suggestion input valiodation failed: %s", err)
	}

	client := github.NewTokenClient(ctx, githubToken)

	title := fmt.Sprintf("%s word: '%s' (via suggestion form)", action, word)
	body := createSuggestionBody(word, language, action, message)
	ir := github.IssueRequest{Title: &title, Body: &body, Labels: &[]string{"enhancement"}}

	gir := github.IssueRequest(ir)
	_, _, err = client.Issues.Create(ctx, "pandorasNox", "lettr", &gir)

	// fmt.Printf("Issue: %v\n", issue)
	// fmt.Printf("Response: %v\n", res)
	// fmt.Printf("Error: %v\n", err)

	return err
}

func validateSuggestionInputs(word, language, action, message string) error {
	//TODO
	return nil
}

func createSuggestionBody(word, language, action, message string) string {
	return fmt.Sprintf(`
Note: This issue was automatically created, but initiated/raised via the lettr website form submit.

action: %s
word: '%s'
language: %s
message:
%s

`, word, language, action, message)
}
