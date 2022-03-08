package ast_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jhump/protocompile/ast"
	"github.com/jhump/protocompile/parser"
	"github.com/jhump/protocompile/reporter"
)

func TestASTRoundTrips(t *testing.T) {
	err := filepath.Walk("../internal/testprotos", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".proto" {
			t.Run(path, func(t *testing.T) {
				data, err := ioutil.ReadFile(path)
				if !assert.Nil(t, err, "%v", err) {
					return
				}
				filename := filepath.Base(path)
				root, err := parser.Parse(filename, bytes.NewReader(data), reporter.NewHandler(nil))
				if !assert.Nil(t, err) {
					return
				}
				var buf bytes.Buffer
				err = printAST(&buf, root)
				if assert.Nil(t, err, "%v", err) {
					// see if file survived round trip!
					assert.Equal(t, string(data), buf.String())
				}
			})
		}
		return nil
	})
	assert.Nil(t, err, "%v", err)
}

// printAST prints the given AST node to the given output. This operation
// basically walks the AST and, for each TerminalNode, prints the node's
// leading comments, leading whitespace, the node's raw text, and then
// any trailing comments. If the given node is a *FileNode, it will then
// also print the file's FinalComments and FinalWhitespace.
func printAST(w *bytes.Buffer, file *ast.FileNode) error {
	err := ast.Walk(file, &ast.SimpleVisitor{
		DoVisitTerminalNode: func(token ast.TerminalNode) error {
			info := file.NodeInfo(token)
			if err := printComments(w, info.LeadingComments()); err != nil {
				return err
			}

			if _, err := w.WriteString(info.LeadingWhitespace()); err != nil {
				return err
			}

			if _, err := w.WriteString(info.RawText()); err != nil {
				return err
			}

			return printComments(w, info.TrailingComments())
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func printComments(w *bytes.Buffer, comments ast.Comments) error {
	for i := 0; i < comments.Len(); i++ {
		comment := comments.Index(i)
		if _, err := w.WriteString(comment.LeadingWhitespace()); err != nil {
			return err
		}
		if _, err := w.WriteString(comment.RawText()); err != nil {
			return err
		}
	}
	return nil
}
