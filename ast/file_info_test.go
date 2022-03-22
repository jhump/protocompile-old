package ast_test

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jhump/protocompile/ast"
	"github.com/jhump/protocompile/parser"
	"github.com/jhump/protocompile/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeadingWhitespace(t *testing.T) {
	const path = "../internal/testprotos/desc_test_leading_whitespace.proto"
	data, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	fileNode, err := parser.Parse(filepath.Base(path), bytes.NewReader(data), reporter.NewHandler(nil))
	require.NoError(t, err)
	require.NoError(
		t,
		ast.Walk(
			fileNode,
			&ast.SimpleVisitor{
				DoVisitMessageNode: func(messageNode *ast.MessageNode) error {
					info := fileNode.NodeInfo(messageNode.Keyword)
					assert.Empty(t, info.LeadingWhitespace(), "%s should not have leading whitespace", messageNode.Name.Val)
					return nil
				},
			},
		),
	)
}
