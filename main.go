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

	flags, ok, err := parseFlags(args)
	if err != nil || !ok {
		flags.flagSet.Usage()
		return err
	}

	sourceFile := flags.flagSet.Arg(0)
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
	session, err := client.Session.New(ctx, opencode.SessionNewParams{
		Directory: opencode.String(flags.dir),
	})
	if err != nil {
		return err
	}

	_, err = client.Session.Get(ctx, session.ID, opencode.SessionGetParams{})
	if err != nil {
		return err
	}

	listenerErrCh := make(chan error, 1)
	go func() {
		listenerErrCh <- agent.ListenForEvents(ctx, client, session.ID)
	}()

	directiveErrCh := make(chan error, 1)
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
				directiveErrCh <- err
				return
			}

			if os.Getenv("SKIP_PROCESS") == "1" {
				print.Warning(os.Stdout, "Skipping processing")
				continue
			}
			rsp, err := client.Session.Prompt(
				ctx,
				session.ID,
				opencode.SessionPromptParams{
					Directory: opencode.String(flags.dir),
					System:    opencode.String(string(systemPrompt)),
					Model:     modelParams,
					Parts: opencode.F(
						[]opencode.SessionPromptParamsPartUnion{
							opencode.TextPartInputParam{
								Type: opencode.F(opencode.TextPartInputType("text")),
								Text: opencode.String(fmt.Sprintf(string(directivePromptFile),
									d.Function,
									sourceFile,
									d.StartLine,
									d.EndLine,
									promptText,
									detectLanguage(sourceFile),
									d.Source,
								)),
							},
						}),
				},
			)
			if err != nil {
				var json []byte
				rsp.UnmarshalJSON(json)
				print.Error(os.Stdout, "err prompting: ", string(json))
				directiveErrCh <- fmt.Errorf("prompting: %w", err)
				return
			}
		}
		print.Success(os.Stdout, "\nAll directives processed. Check filesystem for changes.")
		directiveErrCh <- nil
	}()

	// Wait for completion or cancellation
	select {
	case err := <-directiveErrCh:
		cancel()
		<-listenerErrCh
		return err
	case err := <-listenerErrCh:
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
		<-listenerErrCh
		return ctx.Err()
	}
}

type cliFlags struct {
	host     string
	port     string
	model    string
	provider string
	dir      string

	flagSet *flag.FlagSet
}

func (c cliFlags) BaseURL() string {
	url := c.host
	if c.port != "" {
		url = fmt.Sprintf("%s:%s", url, c.port)
	}
	return url
}

func parseFlags(args []string) (cliFlags, bool, error) {
	flagSet := flag.NewFlagSet("chisel", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: chisel [flags] <file>\n")
		flagSet.PrintDefaults()
	}

	flags := cliFlags{
		host:     "http://localhost",
		port:     "3366",
		model:    "big-pickle",
		provider: "opencode",

		flagSet: flagSet,
	}
	flagSet.StringVar(&flags.dir, "dir", "", "directory to process")
	flagSet.StringVar(&flags.host, "host", flags.host, "opencode server host (including protocol)")
	flagSet.StringVar(&flags.port, "port", flags.port, "opencode server port")
	flagSet.StringVar(&flags.model, "model", flags.model, "model to use")
	flagSet.StringVar(&flags.provider, "provider", flags.provider, "provider to use")

	flagSet.Parse(args)

	if flags.dir == "" {
		return flags, false, fmt.Errorf("--dir flag is required")
	}

	if flagSet.NArg() < 1 {
		return cliFlags{}, false, nil
	}

	return flags, true, nil
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
