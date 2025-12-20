package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

func main() {
	if err := run(context.Background(), os.Args); err != nil {
		log.Fatalf("running: %v", err)
	}
}

func startEventListener(ctx context.Context, client *opencode.Client) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Event stream goroutine panicked: %v\n", r)
		}
	}()

	stream := client.Event.ListStreaming(ctx, opencode.EventListParams{})
	defer stream.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !stream.Next() {
				return
			}

			event := stream.Current()

			switch event.Type {
			case opencode.EventListResponseTypePermissionUpdated:
				fmt.Printf("recieved_event: %s\n", event.Type)
				evt := event.AsUnion().(opencode.EventListResponseEventPermissionUpdated)

				// Use osascript to show a native macOS dialog
				script := `display dialog "Agent is requesting permission to perform an action." with title "Chisel Permission" buttons {"Reject", "Allow Once", "Always"} default button "Always" cancel button "Reject" with icon caution`
				cmd := exec.Command("osascript", "-e", script)
				output, err := cmd.CombinedOutput()

				response := opencode.SessionPermissionRespondParamsResponseReject
				if err == nil {
					outStr := string(output)
					if strings.Contains(outStr, "Always") {
						response = opencode.SessionPermissionRespondParamsResponseAlways
					} else if strings.Contains(outStr, "Allow Once") {
						response = opencode.SessionPermissionRespondParamsResponseOnce
					}
				}

				client.Session.Permissions.Respond(ctx, evt.Properties.SessionID, evt.Properties.ID, opencode.SessionPermissionRespondParams{
					Response: opencode.F(response),
				})
			}
		}
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

	projectSession, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		return err
	}
	fmt.Printf("projectSessionID: %s\n", projectSession.ID)

	go startEventListener(ctx, client)

	// Process each directive with its own child session
	for _, d := range directives {
		// Create a child session for this directive
		childSession, err := client.Session.New(ctx, opencode.SessionNewParams{
			ParentID: opencode.String(projectSession.ID),
		})
		if err != nil {
			return err
		}

		fmt.Printf("Sending directive to agent: %s\n", d.Comment)

		rsp, err := client.Session.Prompt(
			ctx,
			childSession.ID,
			opencode.SessionPromptParams{
				System: opencode.String(`
# Chisel - System Reminder

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
	}

	fmt.Println("\nAll directives processed. Check filesystem for changes.")
	return nil
}
