package directive

import (
	"fmt"
	"log"
	"strings"

	ts "github.com/tree-sitter/go-tree-sitter"
	ts_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// aiCommentQuery matches comments that start with "// @ai".
const aiCommentQuery = `((comment) @ai.comment (#match? @ai.comment "^//\\s*@ai"))`

// functionKinds defines AST node types that represent function-like constructs.
var functionKinds = map[string]bool{
	"function_declaration": true,
	"method_declaration":   true,
	"func_literal":         true,
}

// AIDirective represents an @ai comment and its enclosing function context.
type AIDirective struct {
	Comment      string
	Function     string
	Source       string
	StartLine    uint
	EndLine      uint
	StartByte    uint
	EndByte      uint
	CommentStart uint
	CommentEnd   uint
}

// Parser extracts AI directives from Go source code using tree-sitter.
type Parser struct {
	language *ts.Language
}

// NewParser creates a new Parser configured for Go source code.
func NewParser() *Parser {
	return &Parser{
		language: ts.NewLanguage(ts_go.Language()),
	}
}

func (d *AIDirective) Prompt() (string, error) {
	lines := strings.Split(d.Comment, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimLeft(line, " \t")
		line = strings.TrimPrefix(line, "// ")
		line = strings.TrimPrefix(line, "@ai ")
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n"), nil
}

// Parse extracts all AI directives from the given Go source code.
func (p *Parser) Parse(code []byte) ([]AIDirective, error) {
	log.Printf("Parsing %d bytes of code", len(code))

	parser := ts.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(p.language); err != nil {
		log.Printf("Error setting language: %v", err)
		return nil, fmt.Errorf("setting language: %w", err)
	}

	tree := parser.Parse(code, nil)
	defer tree.Close()

	directives, err := p.extractDirectives(code, tree.RootNode())
	if err != nil {
		log.Printf("Error extracting directives: %v", err)
		return nil, err
	}

	log.Printf("Found %d AI directives", len(directives))
	return directives, nil
}

// extractDirectives runs the query and builds the directive list.
func (p *Parser) extractDirectives(code []byte, root *ts.Node) ([]AIDirective, error) {
	query, err := ts.NewQuery(p.language, aiCommentQuery)
	if err != nil {
		return nil, fmt.Errorf("creating query: %w", err)
	}
	defer query.Close()

	cursor := ts.NewQueryCursor()
	defer cursor.Close()

	captures := cursor.Captures(query, root, code)
	var directives []AIDirective

	for {
		match, _ := captures.Next()
		if match == nil {
			break
		}

		// Query guarantees at least one capture; skip defensively if empty.
		if len(match.Captures) == 0 {
			continue
		}

		commentNode := match.Captures[0].Node
		funcNode := findAssociatedFunction(&commentNode)
		if funcNode == nil {
			continue
		}

		commentText, commentStart, commentEnd := collectCommentBlock(code, &commentNode)
		directives = append(directives, AIDirective{
			Comment:      commentText,
			Function:     extractFunctionName(code, funcNode),
			Source:       extractFunctionSource(code, funcNode),
			StartLine:    funcNode.StartPosition().Row + 1,
			EndLine:      funcNode.EndPosition().Row + 1,
			StartByte:    funcNode.StartByte(),
			EndByte:      funcNode.EndByte(),
			CommentStart: commentStart,
			CommentEnd:   commentEnd,
		})
	}

	return directives, nil
}

// findAssociatedFunction finds the function associated with a comment node.
// It first walks up the AST for comments inside functions, then checks
// following siblings for doc-style comments that precede a function.
func findAssociatedFunction(n *ts.Node) *ts.Node {
	// First, check if we're inside a function (comment is within function body).
	for p := n.Parent(); p != nil; p = p.Parent() {
		if functionKinds[p.Kind()] {
			return p
		}
	}

	// Otherwise, check if a function follows this comment (doc-style comment).
	// Skip over any intervening comments to find the next non-comment sibling.
	for sib := n.NextSibling(); sib != nil; sib = sib.NextSibling() {
		if sib.Kind() == "comment" {
			continue
		}
		if functionKinds[sib.Kind()] {
			return sib
		}
		break
	}

	return nil
}

// extractFunctionName returns the name of a function node, or "<anonymous>"
// for closures/literals that have no name.
func extractFunctionName(code []byte, funcNode *ts.Node) string {
	nameNode := funcNode.ChildByFieldName("name")
	if nameNode == nil {
		return "<anonymous>"
	}
	return string(code[nameNode.StartByte():nameNode.EndByte()])
}

// extractFunctionSource returns the full source code of a function node.
func extractFunctionSource(code []byte, funcNode *ts.Node) string {
	return string(code[funcNode.StartByte():funcNode.EndByte()])
}

// collectCommentBlock gathers a contiguous block of comments around an @ai comment.
// It collects both preceding and following sibling comments to capture the full context.
func collectCommentBlock(code []byte, n *ts.Node) (string, uint, uint) {
	start := n.StartByte()
	end := n.EndByte()

	// Extend backward to include preceding contiguous comments.
	for sib := n.PrevSibling(); sib != nil; sib = sib.PrevSibling() {
		if sib.Kind() != "comment" {
			break
		}
		start = sib.StartByte()
	}

	// Extend forward to include following contiguous comments.
	for sib := n.NextSibling(); sib != nil; sib = sib.NextSibling() {
		if sib.Kind() != "comment" {
			break
		}
		end = sib.EndByte()
	}

	raw := string(code[start:end])

	// Normalize each line by trimming leading whitespace.
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, " \t")
	}

	return strings.Join(lines, "\n"), start, end
}
