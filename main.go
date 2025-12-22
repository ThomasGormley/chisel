package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
	"github.com/thomasgormley/chisel/internal/agent"
	"github.com/thomasgormley/chisel/internal/directive"
	"github.com/thomasgormley/chisel/internal/print"
)

//go:embed prompts/system.md
var systemPrompt []byte

//go:embed prompts/directive-context.md
var directivePromptFile []byte

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		print.Errorf(os.Stderr, "error running CLI: %s\n", err)
	}

	print.Info(os.Stdout, "Press Enter to exit...\n")
	var input string
	fmt.Scanln(&input)
}

func run(ctx context.Context, args []string) error {
	mainCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx, stop := signal.NotifyContext(mainCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	flagSet := flag.NewFlagSet("chisel", flag.ExitOnError)

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: chisel [flags] <file>\n")
		flagSet.PrintDefaults()
	}
	flags := parseFlags(flagSet, args)
	if flagSet.NArg() < 1 {
		flagSet.Usage()
		return nil
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
		print.Warning(os.Stdout, "No @ai directives found. To apply a directive, add a comment like // @ai <instruction> in your code.")
		return nil
	}

	client := opencode.NewClient(option.WithBaseURL(flags.BaseURL()))
	session, err := client.Session.New(ctx, opencode.SessionNewParams{})
	if err != nil {
		return err
	}

	print.Info(os.Stdout, "sessionDir: ", session.Directory)

	eventsDone := make(chan error, 1)
	go func() {
		eventsDone <- agent.ListenForEvents(ctx, client, session.ID, session.Directory)
	}()

	// Process directives
	processDone := make(chan error, 1)
	go func() {
		for _, d := range directives {
			modelParams := opencode.F(opencode.SessionPromptParamsModel{
				ModelID:    opencode.String(flags.model),
				ProviderID: opencode.String(flags.provider),
			})
			print.Info(os.Stdout, "Processing directive in function:", d.Function)
			print.Info(os.Stdout, "->", modelParams.Value.ProviderID.String(), "/", modelParams.Value.ModelID.String())
			promptText, err := d.Prompt()
			if err != nil {
				processDone <- err
				return
			}

			if os.Getenv("SKIP_PROCESS") == "1" {
				// @ai skip processing
				print.Warning(os.Stdout, "Skipping processing")
				continue
			}
			_, err = client.Session.Prompt(
				ctx,
				session.ID,
				opencode.SessionPromptParams{
					System: opencode.String(string(systemPrompt)),
					Model:  modelParams,
					Parts: opencode.F(
						[]opencode.SessionPromptParamsPartUnion{
							opencode.TextPartInputParam{
								Type: opencode.F(opencode.TextPartInputType("text")),
								Text: opencode.String(fmt.Sprintf(string(directivePromptFile),
									d.Function,
									sourceFile,
									promptText,
									detectLanguage(sourceFile),
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
		print.Success(os.Stdout, "\nAll directives processed. Check filesystem for changes.")
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
		print.Warning(os.Stdout, print.Wrap("Shutting down, aborting client session..."))
		abortRsp, err := client.Session.Abort(mainCtx, session.ID, opencode.SessionAbortParams{})
		if err != nil {
			print.Warning(os.Stdout, print.Wrap("Failed to abort client session:", err.Error()))
		} else if abortRsp == nil || !*abortRsp {
			print.Warning(os.Stdout, print.Wrap("Client session abort did not confirm success."))
		} else {
			print.Info(os.Stdout, print.Wrap("Client session aborted successfully."))
		}
		<-eventsDone
		return ctx.Err()
	}
}

type cliFlags struct {
	host     string
	port     string
	model    string
	provider string
}

func (c cliFlags) BaseURL() string {
	url := c.host
	if c.port != "" {
		url = fmt.Sprintf("%s:%s", url, c.port)
	}
	return url
}

func parseFlags(flagSet *flag.FlagSet, args []string) cliFlags {
	flags := cliFlags{
		host:     "http://localhost",
		port:     "3366",
		model:    "big-pickle",
		provider: "opencode",
	}
	flagSet.StringVar(&flags.host, "host", flags.host, "opencode server host (including protocol)")
	flagSet.StringVar(&flags.port, "port", flags.port, "opencode server port")
	flagSet.StringVar(&flags.model, "model", flags.model, "model to use")
	flagSet.StringVar(&flags.provider, "provider", flags.provider, "provider to use")

	flagSet.Parse(args)

	return flags
}

func detectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".go":
		return "go"
	default:
		return ""
	}
}
