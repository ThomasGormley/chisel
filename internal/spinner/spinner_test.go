package spinner

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thomasgormley/chisel/internal/print"
)

func TestWith_Success(t *testing.T) {
	var buf bytes.Buffer
	message := "Test operation"
	successMsg := "Operation completed successfully"

	err := With(message, func() error {
		time.Sleep(50 * time.Millisecond) // Simulate work
		return nil
	}, WithWriter(&buf), WithSuccessMessage(successMsg))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected some output, got empty string")
	}
}

func TestWith_Error(t *testing.T) {
	var buf bytes.Buffer
	message := "Test operation"
	failureMsg := "Operation failed"
	testErr := errors.New("test error")

	err := With(message, func() error {
		time.Sleep(50 * time.Millisecond) // Simulate work
		return testErr
	}, WithWriter(&buf), WithFailureMessage(failureMsg))

	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected some output, got empty string")
	}
}

func TestWithContext_Success(t *testing.T) {
	var buf bytes.Buffer
	message := "Test operation with context"
	successMsg := "Context operation completed"

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := WithContext(ctx, message, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond) // Simulate work
		return nil
	}, WithWriter(&buf), WithSuccessMessage(successMsg))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected some output, got empty string")
	}
}

func TestWithContext_ContextTimeout(t *testing.T) {
	var buf bytes.Buffer
	message := "Test operation with timeout"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := WithContext(ctx, message, func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond) // Simulate work that exceeds timeout
		return nil
	}, WithWriter(&buf))

	if err == nil {
		t.Error("Expected context timeout error, got nil")
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected some output, got empty string")
	}
}

func TestStart_ManualControl(t *testing.T) {
	var buf bytes.Buffer
	message := "Manual spinner test"

	pin, cancel := Start(message, WithWriter(&buf))
	defer cancel()

	// Let spinner run briefly
	time.Sleep(50 * time.Millisecond)

	// Test message update
	pin.UpdateMessage("Updated message")
	time.Sleep(50 * time.Millisecond)

	// Stop with success
	pin.Stop("Manual success")

	output := buf.String()
	if output == "" {
		t.Error("Expected some output, got empty string")
	}
}

func TestStart_ManualControl_Failure(t *testing.T) {
	var buf bytes.Buffer
	message := "Manual spinner failure test"

	pin, cancel := Start(message, WithWriter(&buf))
	defer cancel()

	// Let spinner run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop with failure
	pin.Fail("Manual failure")

	output := buf.String()
	if output == "" {
		t.Error("Expected some output, got empty string")
	}
}

func TestOptions(t *testing.T) {
	var buf bytes.Buffer
	message := "Options test"

	err := With(message, func() error {
		return nil
	},
		WithWriter(&buf),
		WithSpinnerColor(print.ColorYellow),
		WithTextColor(print.ColorBlue),
		WithDoneSymbol('✓'),
		WithDoneSymbolColor(print.ColorGreen),
		WithFailSymbol('✗'),
		WithFailSymbolColor(print.ColorRed),
		WithSuccessMessage("Custom success"),
	)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected some output, got empty string")
	}
}

func TestDefaultConfig(t *testing.T) {
	var buf bytes.Buffer
	message := "Default config test"

	err := With(message, func() error {
		return nil
	}, WithWriter(&buf))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected some output, got empty string")
	}
}
