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
	if len(args) < 2 {
		return fmt.Errorf("usage: chisel <file>")
	}
	sourceFile := args[1]

	file, err := os.ReadFile(sourceFile)
	if err != nil {
		return err
	}
	parser := NewParser()
	directives, err := parser.Parse(file)
	if err != nil {
		return err
	}

	if len(directives) == 0 {
		fmt.Println("No @ai directives found.")
		return nil
	}

	client := opencode.NewClient(option.WithBaseURL("http://localhost:3366"))
	session, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		return err
	}

	// We only handle the first directive for now
	d := directives[0]

	fmt.Printf("Sending directive to agent: %s\n", d.Comment)

	rsp, err := client.Session.Prompt(
		ctx,
		session.ID,
		opencode.SessionPromptParams{
			NoReply: opencode.Bool(true),
			Model: opencode.F(opencode.SessionPromptParamsModel{
				ModelID:    opencode.String("big-pickle"),
				ProviderID: opencode.String("opencode"),
			}),
			Parts: opencode.F(
				[]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type:      opencode.F(opencode.TextPartInputType("text")),
						Synthetic: opencode.Bool(true),
						Text: opencode.String(`
# Chisel Mode - System Reminder

CRITICAL: Modification mode ACTIVE. You are authorized to use your tools to fulfill the @ai directive.

---

## Responsibility
Your responsibility is to fulfill the @ai directive provided below.
1. **Analyze** the provided source code and the directive.
2. **Apply** the requested changes using your tools.
3. **Remove** the @ai comment as part of your edit.

---

## Important
You have been provided with the exact source of the function and its location. Use this to understand the context, but use your tools to apply the change to the file.
`),
					},
				}),
		},
	)

	if err != nil {
		return err
	}

	rsp, err = client.Session.Prompt(
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
						Text: opencode.String(fmt.Sprintf(
							"Directive: %s\nFile: %s\nFunction: %s\nLine: %d\n\nSource:\n%s",
							d.Comment,
							sourceFile,
							d.Function,
							d.StartLine,
							d.Source,
						)),
					},
				}),
		},
	)

	if err != nil {
		return err
	}

	fmt.Println("\n--- Agent Response Log ---")
	for _, part := range rsp.Parts {
		fmt.Printf("[%s]", part.Type)
		if part.Text != "" {
			fmt.Printf(" %s", part.Text)
		}
		if part.Tool != "" {
			fmt.Printf(" Tool: %s", part.Tool)
		}
		if part.Reason != "" {
			fmt.Printf(" Reason: %s", part.Reason)
		}
		fmt.Println()
	}
	fmt.Println("--------------------------")

	fmt.Println("\nDirective processed. Check filesystem for changes.")
	return nil
}
