package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v73/github"
)

func main() {
	ctx := context.Background()
	secret := os.Args[1]

	client := github.NewTokenClient(ctx, secret)

	issueTitle := "new word suggestion test"
	ir := github.IssueRequest{Title: &issueTitle}

	issue, res, err := client.Issues.Create(ctx, "pandorasNox", "lettr", &ir)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	fmt.Printf("Issue: %v\n", issue)
	fmt.Printf("Response: %v\n", res)
}
