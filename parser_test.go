package main

import (
	"testing"
)

func TestParserParse(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []AIDirective
	}{
		{
			name: "simple function with @ai comment",
			code: `package main

func doSomething() error {
	// @ai implement this
	return nil
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai implement this",
					Function:  "doSomething",
					Source:    "func doSomething() error {\n\t// @ai implement this\n\treturn nil\n}",
					StartLine: 3,
					EndLine:   6,
				},
			},
		},
		{
			name: "multiline @ai comment block",
			code: `package main

func process() {
	// @ai implement the function
	// with multiple lines
	// of instructions
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai implement the function\n// with multiple lines\n// of instructions",
					Function:  "process",
					Source:    "func process() {\n\t// @ai implement the function\n\t// with multiple lines\n\t// of instructions\n}",
					StartLine: 3,
					EndLine:   7,
				},
			},
		},
		{
			name: "method declaration",
			code: `package main

func (s *Server) handleRequest() error {
	// @ai handle the request
	return nil
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai handle the request",
					Function:  "handleRequest",
					Source:    "func (s *Server) handleRequest() error {\n\t// @ai handle the request\n\treturn nil\n}",
					StartLine: 3,
					EndLine:   6,
				},
			},
		},
		{
			name: "anonymous function (closure)",
			code: `package main

func outer() {
	fn := func() {
		// @ai implement closure
	}
	_ = fn
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai implement closure",
					Function:  "<anonymous>",
					Source:    "func() {\n\t\t// @ai implement closure\n\t}",
					StartLine: 4,
					EndLine:   6,
				},
			},
		},
		{
			name: "doc-style comment before function",
			code: `package main

// @ai implement this function
func calculate() int {
	return 0
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai implement this function",
					Function:  "calculate",
					Source:    "func calculate() int {\n\treturn 0\n}",
					StartLine: 4,
					EndLine:   6,
				},
			},
		},
		{
			name: "doc-style comment block before function",
			code: `package main

// Some context here
// @ai implement with care
// more details follow
func important() {
}
`,
			expected: []AIDirective{
				{
					Comment:   "// Some context here\n// @ai implement with care\n// more details follow",
					Function:  "important",
					Source:    "func important() {\n}",
					StartLine: 6,
					EndLine:   7,
				},
			},
		},
		{
			name: "doc-style comment before method",
			code: `package main

// @ai fix this method
func (c *Client) connect() error {
	return nil
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai fix this method",
					Function:  "connect",
					Source:    "func (c *Client) connect() error {\n\treturn nil\n}",
					StartLine: 4,
					EndLine:   6,
				},
			},
		},
		{
			name: "multiple functions with @ai comments",
			code: `package main

func first() {
	// @ai implement first
}

func second() {
	// @ai implement second
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai implement first",
					Function:  "first",
					Source:    "func first() {\n\t// @ai implement first\n}",
					StartLine: 3,
					EndLine:   5,
				},
				{
					Comment:   "// @ai implement second",
					Function:  "second",
					Source:    "func second() {\n\t// @ai implement second\n}",
					StartLine: 7,
					EndLine:   9,
				},
			},
		},
		{
			name: "no @ai comments",
			code: `package main

func regular() {
	// just a normal comment
	return
}
`,
			expected: nil,
		},
		{
			name: "@ai comment not in a function",
			code: `package main

// @ai orphan comment

var x = 1
`,
			expected: nil,
		},
		{
			name: "@ai with varying whitespace",
			code: `package main

func spaced() {
	//  @ai extra space before directive
}
`,
			expected: []AIDirective{
				{
					Comment:   "//  @ai extra space before directive",
					Function:  "spaced",
					Source:    "func spaced() {\n\t//  @ai extra space before directive\n}",
					StartLine: 3,
					EndLine:   5,
				},
			},
		},
		{
			name: "nested closure with @ai",
			code: `package main

func outer() {
	inner := func() {
		deep := func() {
			// @ai deeply nested
		}
		_ = deep
	}
	_ = inner
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai deeply nested",
					Function:  "<anonymous>",
					Source:    "func() {\n\t\t\t// @ai deeply nested\n\t\t}",
					StartLine: 5,
					EndLine:   7,
				},
			},
		},
		{
			name: "mixed: doc comment and body comment",
			code: `package main

// @ai doc comment
func mixed() {
	// @ai body comment
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai doc comment",
					Function:  "mixed",
					Source:    "func mixed() {\n\t// @ai body comment\n}",
					StartLine: 4,
					EndLine:   6,
				},
				{
					Comment:   "// @ai body comment",
					Function:  "mixed",
					Source:    "func mixed() {\n\t// @ai body comment\n}",
					StartLine: 4,
					EndLine:   6,
				},
			},
		},
		{
			name: "function with parameters and return types",
			code: `package main

func transform(input string, count int) (string, error) {
	// @ai transform the input
	return "", nil
}
`,
			expected: []AIDirective{
				{
					Comment:   "// @ai transform the input",
					Function:  "transform",
					Source:    "func transform(input string, count int) (string, error) {\n\t// @ai transform the input\n\treturn \"\", nil\n}",
					StartLine: 3,
					EndLine:   6,
				},
			},
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directives, err := parser.Parse([]byte(tt.code))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(directives) != len(tt.expected) {
				t.Fatalf("expected %d directives, got %d", len(tt.expected), len(directives))
			}

			for i, exp := range tt.expected {
				got := directives[i]

				if got.Comment != exp.Comment {
					t.Errorf("directive[%d].Comment:\n  expected: %q\n  got:      %q", i, exp.Comment, got.Comment)
				}
				if got.Function != exp.Function {
					t.Errorf("directive[%d].Function: expected %q, got %q", i, exp.Function, got.Function)
				}
				if got.Source != exp.Source {
					t.Errorf("directive[%d].Source:\n  expected: %q\n  got:      %q", i, exp.Source, got.Source)
				}
				if got.StartLine != exp.StartLine {
					t.Errorf("directive[%d].StartLine: expected %d, got %d", i, exp.StartLine, got.StartLine)
				}
				if got.EndLine != exp.EndLine {
					t.Errorf("directive[%d].EndLine: expected %d, got %d", i, exp.EndLine, got.EndLine)
				}
			}
		})
	}
}
