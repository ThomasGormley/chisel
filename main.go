package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/thomasgormley/chisel/internal/agent"
	"github.com/thomasgormley/chisel/internal/directive"
)

//go:embed prompts/system.md
var systemPrompt []byte

//go:embed prompts/directive-context.md
var directivePromptFile []byte

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Print("error... " + err.Error())
		var input string
		fmt.Scanln(&input)
		log.Fatalf("running: %v", err)
	}

	fmt.Print("Press Enter to exit...")
	var input string
	fmt.Scanln(&input)
}

func run(ctx context.Context, args []string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Parse flags
	flagSet := flag.NewFlagSet("chisel", flag.ExitOnError)
	host := flagSet.String("host", "http://localhost", "opencode server host (including protocol)")
	port := flagSet.String("port", "3366", "opencode server port")
	flagSet.Parse(args)

	if flagSet.NArg() < 1 {
		return fmt.Errorf("usage: chisel [flags] <file>")
	}
	sourceFile := flagSet.Arg(0)

	baseURL := *host
	if *port != "" {
		baseURL = fmt.Sprintf("%s:%s", baseURL, *port)
	}

	// Read and parse directives
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

	// Setup client and session
	client := opencode.NewClient(option.WithBaseURL(baseURL))
	mainSession, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		return err
	}

	// Start event listener
	eventsDone := make(chan error, 1)
	go func() {
		eventsDone <- agent.ListenForEvents(ctx, client)
	}()

	// Process directives
	processDone := make(chan error, 1)
	go func() {
		for _, d := range directives {
			log.Printf("Processing directive: %s", d.Function)
			promptText, err := d.Prompt()
			if err != nil {
				processDone <- err
				return
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
				processDone <- err
				return
			}
		}
		fmt.Println("\nAll directives processed. Check filesystem for changes.")
		processDone <- nil
	}()

	// Wait for completion or cancellation
	select {
	case err := <-processDone:
		cancel()
		<-eventsDone
		return err
	case err := <-eventsDone:
		cancel()
		return fmt.Errorf("event stream error: %w", err)
	case <-ctx.Done():
		fmt.Println("Shutting down, aborting client session...")
		client.Session.Abort(ctx, mainSession.ID, opencode.SessionAbortParams{})
		<-eventsDone
		return ctx.Err()
	}
}
