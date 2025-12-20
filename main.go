package main

import (
	"fmt"
	"log"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"
)

func main() {
	code := []byte(`package main

import (
	"fmt"
	"log"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"
	ts_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func doSomething(ctx context.Context, s string) error {

	fnc := func() {
		// @ai implement the function
		// Now we have multilines
		// And one more
	}

	return nil
}

func doAnotherThing(ctx context.Context, s string) error {
	// @ai can you also do this?

	return nil
}

// Some context here
// @ai this is a method directive
// with more details
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) error {
	return nil
}
`)

	parser := NewParser()
	directives, err := parser.Parse(code)
	if err != nil {
		log.Fatalf("parsing: %v", err)
	}

	for _, d := range directives {
		fmt.Printf("Function: %s (lines %d-%d)\n", d.Function, d.StartLine, d.EndLine)
		fmt.Printf("Comment:\n%s\n\n", d.Comment)
	}
}

// printTree prints the syntax tree rooted at n with indentation (for debugging).
func printTree(n *ts.Node, depth int) {
	if n == nil {
		return
	}

	indent := strings.Repeat("  ", depth)
	start := n.StartPosition()
	end := n.EndPosition()

	fmt.Printf("%s%s [%d:%d - %d:%d]\n",
		indent,
		n.Kind(),
		start.Row+1, start.Column+1,
		end.Row+1, end.Column+1,
	)

	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		printTree(child, depth+1)
	}
}
