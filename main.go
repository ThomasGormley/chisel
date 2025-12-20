package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

func main() {
	if err := run(context.Background(), os.Args); err != nil {
		log.Fatalf("running: %v", err)
	}
}

func run(ctx context.Context, args []string) error {

	sourceFile := args[1]
	fmt.Println(sourceFile)

	file, err := os.ReadFile(sourceFile)
	if err != nil {
		return err
	}
	parser := NewParser()
	directives, err := parser.Parse(file)
	if err != nil {
		return err
	}

	for _, d := range directives {
		fmt.Printf("Function: %s (lines %d-%d, bytes %d-%d)\n", d.Function, d.StartLine, d.EndLine, d.StartByte, d.EndByte)
		fmt.Printf("Comment: (bytes %d-%d)\n%s\n", d.CommentStart, d.CommentEnd, d.Comment)
		fmt.Printf("Source:\n%s\n\n\n", d.Source)
	}

	client := opencode.NewClient(option.WithBaseURL("http://localhost:3366"))
	session, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		return err
	}
	rsp, err := client.Session.Prompt(
		ctx,
		session.ID,
		opencode.SessionPromptParams{
			Model: opencode.F(opencode.SessionPromptParamsModel{
				ModelID:    opencode.String("big-pickle"),
				ProviderID: opencode.String("opencode"),
			}),
			Parts: opencode.F(
				[]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputType("text")),
						Text: opencode.String(fmt.Sprintf("%s \nfile: %s:%d", directives[0].Comment, sourceFile, directives[0].StartLine)),
					},
				}),
		},
	)

	if err != nil {
		return err
	}

	for _, part := range rsp.Parts {
		if part.Type == "text" {
			fmt.Printf("agent_message_chunk\n%s\n", part.Text)
		} else {
			fmt.Printf("%s\n", part.Type)
		}
	}
	return nil
}
