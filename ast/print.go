package ast

import "io"

// Print prints the given AST node to the given output. This operation
// basically walks the AST and, for each TerminalNode, prints the node's
// leading comments, leading whitespace, the node's raw text, and then
// any trailing comments. If the given node is a *FileNode, it will then
// also print the file's FinalComments and FinalWhitespace.
func Print(w io.Writer, node Node) error {
	sw, ok := w.(stringWriter)
	if !ok {
		sw = &strWriter{w}
	}
	err := Walk(node, &SimpleVisitor{
		DoVisitTerminalNode: func(token TerminalNode) error {
			if err := printComments(sw, token.LeadingComments()); err != nil {
				return err
			}

			if _, err := sw.WriteString(token.LeadingWhitespace()); err != nil {
				return err
			}

			if _, err := sw.WriteString(token.RawText()); err != nil {
				return err
			}

			return printComments(sw, token.TrailingComments())
		},
	})
	if err != nil {
		return err
	}

	if file, ok := node.(*FileNode); ok {
		err = printComments(sw, file.FinalComments)
		if err != nil {
			return err
		}
		_, err = sw.WriteString(file.FinalWhitespace)
		return err
	}

	return nil
}

func printComments(sw stringWriter, comments []Comment) error {
	for _, comment := range comments {
		if _, err := sw.WriteString(comment.LeadingWhitespace); err != nil {
			return err
		}
		if _, err := sw.WriteString(comment.Text); err != nil {
			return err
		}
	}
	return nil
}

// many io.Writer impls also provide a string-based method
type stringWriter interface {
	WriteString(s string) (n int, err error)
}

// adapter, in case the given writer does NOT provide a string-based method
type strWriter struct {
	io.Writer
}

func (s *strWriter) WriteString(str string) (int, error) {
	if str == "" {
		return 0, nil
	}
	return s.Write([]byte(str))
}
