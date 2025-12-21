package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

//go:embed prompts/system.md
var systemPromptFile []byte

//go:embed prompts/directive-context.md
var directivePromptFile []byte

func main() {
	if err := run(context.Background(), os.Args); err != nil {
		log.Fatalf("running: %v", err)
	}
}

func run(ctx context.Context, args []string) error {
	var (
		host = flag.String("host", "http://localhost", "opencode server host (including protocol)")
		port = flag.String("port", "3366", "opencode server port")
	)
	flag.Parse()

	if flag.NArg() < 1 {
		return fmt.Errorf("usage: chisel [flags] <file>")
	}
	sourceFile := flag.Arg(0)

	systemPrompt := systemPromptFile

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

	baseURL := *host
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	if *port != "" {
		baseURL = fmt.Sprintf("%s:%s", baseURL, *port)
	}
	client := opencode.NewClient(option.WithBaseURL(baseURL))

	mainSession, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		return err
	}

	go listen(ctx, client)

	// Process each directive with its own child session
	for _, d := range directives {
		// Create a child session for this directive
		childSession, err := client.Session.New(ctx, opencode.SessionNewParams{
			ParentID: opencode.String(mainSession.ID),
		})
		if err != nil {
			return err
		}

		fmt.Printf("Sending directive to agent: %s\n", d.Comment)

		rsp, err := client.Session.Prompt(
			ctx,
			childSession.ID,
			opencode.SessionPromptParams{
				System: opencode.String(string(systemPrompt)),
				Model: opencode.F(opencode.SessionPromptParamsModel{
					ModelID:    opencode.String("big-pickle"),
					ProviderID: opencode.String("opencode"),
				}),
				Parts: opencode.F(
					[]opencode.SessionPromptParamsPartUnion{
						opencode.TextPartInputParam{
							Type: opencode.F(opencode.TextPartInputType("text")),

							Text: opencode.String(fmt.Sprintf(string(directivePromptFile),
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

	fmt.Print("Press Enter to exit...")
	var input string
	fmt.Scanln(&input)
	return nil
}
