package spinner

import (
	"context"
	"io"

	"github.com/thomasgormley/chisel/internal/print"
)

// Option is a functional option for configuring spinner behavior
type Option func(*config)

type config struct {
	writer          io.Writer
	successMessage  string
	failureMessage  string
	spinnerColor    print.Color
	textColor       print.Color
	doneSymbol      rune
	doneSymbolColor print.Color
	failSymbol      rune
	failSymbolColor print.Color
}

// WithWriter sets the io.Writer for spinner output
func WithWriter(w io.Writer) Option {
	return func(c *config) {
		c.writer = w
	}
}

// WithSuccessMessage sets the message displayed on successful completion
func WithSuccessMessage(message string) Option {
	return func(c *config) {
		c.successMessage = message
	}
}

// WithFailureMessage sets the message displayed on failure
func WithFailureMessage(message string) Option {
	return func(c *config) {
		c.failureMessage = message
	}
}

// WithSpinnerColor sets the color of the spinning animation
func WithSpinnerColor(color print.Color) Option {
	return func(c *config) {
		c.spinnerColor = color
	}
}

// WithTextColor sets the color of the message text
func WithTextColor(color print.Color) Option {
	return func(c *config) {
		c.textColor = color
	}
}

// WithDoneSymbol sets the symbol displayed when the spinner completes
func WithDoneSymbol(symbol rune) Option {
	return func(c *config) {
		c.doneSymbol = symbol
	}
}

// WithDoneSymbolColor sets the color of the completion symbol
func WithDoneSymbolColor(color print.Color) Option {
	return func(c *config) {
		c.doneSymbolColor = color
	}
}

// WithFailSymbol sets the symbol displayed when the spinner fails
func WithFailSymbol(symbol rune) Option {
	return func(c *config) {
		c.failSymbol = symbol
	}
}

// WithFailSymbolColor sets the color of the failure symbol
func WithFailSymbolColor(color print.Color) Option {
	return func(c *config) {
		c.failSymbolColor = color
	}
}

// With runs a function with a spinner, handling success/failure automatically
func With(message string, fn func() error, opts ...Option) error {
	return WithContext(context.Background(), message, func(ctx context.Context) error {
		return fn()
	}, opts...)
}

// WithContext runs a function with a spinner using context, handling success/failure automatically
func WithContext(ctx context.Context, message string, fn func(context.Context) error, opts ...Option) error {
	cfg := &config{
		writer:          nil, // Will use print.NewPin default (os.Stdout)
		successMessage:  "",
		failureMessage:  "",
		spinnerColor:    print.ColorCyan,
		textColor:       print.ColorDefault,
		doneSymbol:      '✓',
		doneSymbolColor: print.ColorGreen,
		failSymbol:      '✗',
		failSymbolColor: print.ColorRed,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Build print.Pin options
	pinOpts := []print.Option{
		print.WithSpinnerColor(cfg.spinnerColor),
		print.WithTextColor(cfg.textColor),
		print.WithDoneSymbol(cfg.doneSymbol),
		print.WithDoneSymbolColor(cfg.doneSymbolColor),
		print.WithFailSymbol(cfg.failSymbol),
		print.WithFailSymbolColor(cfg.failSymbolColor),
	}

	if cfg.writer != nil {
		pinOpts = append(pinOpts, print.WithWriter(cfg.writer))
	}

	pin := print.NewPin(message, pinOpts...)
	cancel := pin.Start(ctx)
	defer cancel()

	// Check if context is already cancelled
	if ctx.Err() != nil {
		if cfg.failureMessage != "" {
			pin.Fail(cfg.failureMessage)
		} else {
			pin.Fail("Operation failed")
		}
		return ctx.Err()
	}

	err := fn(ctx)

	if err != nil {
		if cfg.failureMessage != "" {
			pin.Fail(cfg.failureMessage)
		} else {
			pin.Fail("Operation failed")
		}
		return err
	}

	// Check if context was cancelled during function execution
	if ctx.Err() != nil {
		if cfg.failureMessage != "" {
			pin.Fail(cfg.failureMessage)
		} else {
			pin.Fail("Operation failed")
		}
		return ctx.Err()
	}

	if cfg.successMessage != "" {
		pin.Stop(cfg.successMessage)
	} else {
		pin.Stop("Operation completed")
	}

	return nil
}

// Start creates and starts a spinner for manual control
func Start(message string, opts ...Option) (*print.Pin, context.CancelFunc) {
	cfg := &config{
		writer:          nil, // Will use print.NewPin default (os.Stdout)
		spinnerColor:    print.ColorCyan,
		textColor:       print.ColorDefault,
		doneSymbol:      '✓',
		doneSymbolColor: print.ColorGreen,
		failSymbol:      '✗',
		failSymbolColor: print.ColorRed,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Build print.Pin options
	pinOpts := []print.Option{
		print.WithSpinnerColor(cfg.spinnerColor),
		print.WithTextColor(cfg.textColor),
		print.WithDoneSymbol(cfg.doneSymbol),
		print.WithDoneSymbolColor(cfg.doneSymbolColor),
		print.WithFailSymbol(cfg.failSymbol),
		print.WithFailSymbolColor(cfg.failSymbolColor),
	}

	if cfg.writer != nil {
		pinOpts = append(pinOpts, print.WithWriter(cfg.writer))
	}

	pin := print.NewPin(message, pinOpts...)
	cancelFunc := pin.Start(context.Background())

	return pin, cancelFunc
}
