package print

import (
	"bytes"
	"testing"
)

func TestColorize(t *testing.T) {
	text := "test message"
	colored := colorize(SuccessColor, text)
	expected := "\x1b[32mtest message\x1b[0m"

	if colored != expected {
		t.Errorf("Expected %q, got %q", expected, colored)
	}
}

func TestSuccess(t *testing.T) {
	var buf bytes.Buffer
	Success(&buf, "test success")

	output := buf.String()
	expected := "\x1b[32mtest success\x1b[0m\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestSuccessVariadic(t *testing.T) {
	var buf bytes.Buffer
	Success(&buf, Tick, "test success")

	output := buf.String()
	expected := "\x1b[32m✓ test success\x1b[0m\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	Error(&buf, "test error")

	output := buf.String()
	expected := "\x1b[31mtest error\x1b[0m\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestErrorVariadic(t *testing.T) {
	var buf bytes.Buffer
	Error(&buf, Cross, "test error")

	output := buf.String()
	expected := "\x1b[31m✗ test error\x1b[0m\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestWarning(t *testing.T) {
	var buf bytes.Buffer
	Warning(&buf, "test warning")

	output := buf.String()
	expected := "\x1b[33mtest warning\x1b[0m\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestWarningVariadic(t *testing.T) {
	var buf bytes.Buffer
	Warning(&buf, WarningSym, "test warning")

	output := buf.String()
	expected := "\x1b[33m⚠ test warning\x1b[0m\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	Info(&buf, "test info")

	output := buf.String()
	expected := "test info\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestInfoVariadic(t *testing.T) {
	var buf bytes.Buffer
	Info(&buf, InfoSym, "test info")

	output := buf.String()
	expected := "ℹ test info\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestNote(t *testing.T) {
	var buf bytes.Buffer
	Note(&buf, "test note")

	output := buf.String()
	expected := "\x1b[36mtest note\x1b[0m\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestNoteVariadic(t *testing.T) {
	var buf bytes.Buffer
	Note(&buf, InfoSym, "test note")

	output := buf.String()
	expected := "\x1b[36mℹ test note\x1b[0m\n"

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestColorSuccess(t *testing.T) {
	result := ColorSuccess("success")
	expected := "\x1b[32msuccess\x1b[0m"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestColorError(t *testing.T) {
	result := ColorError("error")
	expected := "\x1b[31merror\x1b[0m"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestColorWarning(t *testing.T) {
	result := ColorWarning("warning")
	expected := "\x1b[33mwarning\x1b[0m"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestColorNote(t *testing.T) {
	result := ColorNote("note")
	expected := "\x1b[36mnote\x1b[0m"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWrap(t *testing.T) {
	result := Wrap("test")
	expected := "\ntest\n"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWrapVariadic(t *testing.T) {
	result := Wrap("test", "multiple", "parts")
	expected := "\ntest multiple parts\n"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWrapTop(t *testing.T) {
	result := WrapTop("test")
	expected := "\ntest"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWrapTopVariadic(t *testing.T) {
	result := WrapTop("test", "multiple")
	expected := "\ntest multiple"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWrapBottom(t *testing.T) {
	result := WrapBottom("test")
	expected := "test\n"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWrapBottomVariadic(t *testing.T) {
	result := WrapBottom("test", "multiple")
	expected := "test multiple\n"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWrapMulti(t *testing.T) {
	result := WrapMulti(2, "test")
	expected := "\n\ntest\n\n"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestWrapMultiVariadic(t *testing.T) {
	result := WrapMulti(2, "test", "multiple")
	expected := "\n\ntest multiple\n\n"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestInfof(t *testing.T) {
	var buf bytes.Buffer
	Infof(&buf, "test %s", "info")

	output := buf.String()
	expected := "test info" // No newline

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestSuccessf(t *testing.T) {
	var buf bytes.Buffer
	Successf(&buf, "test %s", "success")

	output := buf.String()
	expected := "\x1b[32mtest success\x1b[0m" // No newline

	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}
