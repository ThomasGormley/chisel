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
	session, err := client.Session.New(ctx, opencode.SessionNewParams{
		// Directory: opencode.String("/Users/thomasgormley/dev/dev-cli-go"),
	})
	if err != nil {
		return err
	}

	// We only handle the first directive for now
	d := directives[0]

	go func() {
		stream := client.Event.ListStreaming(ctx, opencode.EventListParams{})

		for stream.Next() {
			event := stream.Current()

			fmt.Printf("recieved_event: %s\n", event.Type)
			switch event.Type {
			case opencode.EventListResponseTypePermissionUpdated:
				evt := event.AsUnion().(opencode.EventListResponseEventPermissionUpdated)
				client.Session.Permissions.Respond(ctx, evt.Properties.SessionID, evt.Properties.ID, opencode.SessionPermissionRespondParams{
					Response: opencode.F(opencode.SessionPermissionRespondParamsResponseAlways),
				})
				fmt.Printf("handled_event: %s\n", event.Type)
			}
		}
	}()

	fmt.Printf("Sending directive to agent: %s\n", d.Comment)

	rsp, err := client.Session.Prompt(
		ctx,
		session.ID,
		opencode.SessionPromptParams{
			System: opencode.String(`
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
			Model: opencode.F(opencode.SessionPromptParamsModel{
				ModelID:    opencode.String("big-pickle"),
				ProviderID: opencode.String("opencode"),
			}),
			Parts: opencode.F(
				[]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputType("text")),
						Text: opencode.String(fmt.Sprintf(`
# Directive Context

<context>
File: %s
Function: %s
Line: %d
</context>

<directive>
%s
</directive>

<source>
%s
</source>
`,
							sourceFile,
							d.Function,
							d.StartLine,
							d.Comment,
							d.Source,
						)),
					},
				}),
		},
	)

	if err != nil {
		return err
	}

	var rspBytes []byte
	if err := rsp.UnmarshalJSON(rspBytes); err != nil {
		return err
	}

	fmt.Printf("%s\n", rspBytes)

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
