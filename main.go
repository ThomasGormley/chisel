package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/thomasgormley/chisel/internal/directive"
)

//go:embed prompts/system.md
var systemPrompt []byte

//go:embed prompts/directive-context.md
var directivePromptFile []byte

func main() {
	// @ai add graceful shutdown
	if err := run(context.Background(), os.Args); err != nil {
		log.Fatalf("running: %v", err)
	}
}

func run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("chisel", flag.ExitOnError)
	var (
		host = flagSet.String("host", "http://localhost", "opencode server host (including protocol)")
		port = flagSet.String("port", "3366", "opencode server port")
	)
	flagSet.Parse(args)

	if flagSet.NArg() < 1 {
		return fmt.Errorf("usage: chisel [flags] <file>")
	}
	sourceFile := flagSet.Arg(0)

	file, err := os.ReadFile(sourceFile)
	if err != nil {
		return err
	}
	parser := directive.NewParser()
	directives, err := parser.Parse(file)
	if err != nil {
		return err
	}

	if len(directives) == 0 {
		fmt.Println("No @ai directives found.")
		return nil
	}

	baseURL := *host
	if *port != "" {
		baseURL = fmt.Sprintf("%s:%s", baseURL, *port)
	}
	client := opencode.NewClient(option.WithBaseURL(baseURL))

	mainSession, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		return err
	}

	go eventListener(ctx, client)

	for _, d := range directives {
		promptText, err := d.Prompt()
		if err != nil {
			return err
		}
		_, err = client.Session.Prompt(
			ctx,
			mainSession.ID,
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
								promptText,
								d.Source,
							)),
						},
					}),
			},
		)

		if err != nil {
			return err
		}
	}

	fmt.Println("\nAll directives processed. Check filesystem for changes.")

	fmt.Print("Press Enter to exit...")
	var input string
	fmt.Scanln(&input)
	return nil
}
