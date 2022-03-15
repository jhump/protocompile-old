package ast_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jhump/protocompile/ast"
	"github.com/jhump/protocompile/parser"
	"github.com/jhump/protocompile/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompactOptionsLeadingComments(t *testing.T) {
	const path = "../internal/testprotos/desc_test_compact_options.proto"
	data, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	fileNode, err := parser.Parse(filepath.Base(path), bytes.NewReader(data), reporter.NewHandler(nil))
	require.NoError(t, err)
	require.NoError(
		t,
		ast.Walk(
			fileNode,
			&ast.SimpleVisitor{
				DoVisitFieldReferenceNode: func(fieldReferenceNode *ast.FieldReferenceNode) error {
					// We're only testing compact options, so we can confidently
					// retrieve the leading comments from the FieldReference's name
					// since it will always be a terminal *IdentNode unless the
					// field reference has a '('.
					info := fileNode.NodeInfo(fieldReferenceNode.Name)
					if fieldReferenceNode.Open != nil {
						// The leading comments will be attached to the '(', if one exists.
						info = fileNode.NodeInfo(fieldReferenceNode.Open)
					}
					name := stringForFieldReference(fieldReferenceNode)
					if assert.Equal(t, 1, info.LeadingComments().Len(), "%s should have a leading comment", name) {
						assert.Equal(
							t,
							fmt.Sprintf("// Leading comment on %s.\n", name),
							info.LeadingComments().Index(0).RawText(),
						)
					}
					return nil
				},
				DoVisitFieldNode: func(fieldNode *ast.FieldNode) error {
					// The fields in these tests always define a label,
					// so we can confidently use it to retrieve the comments.
					info := fileNode.NodeInfo(fieldNode.Label)
					name := fieldNode.Name.Val
					if assert.Equal(t, 1, info.LeadingComments().Len(), "%s should have a leading comment", name) {
						assert.Equal(
							t,
							fmt.Sprintf("// Leading comment on %s.\n", name),
							info.LeadingComments().Index(0).RawText(),
						)
					}
					return nil
				},
			},
		),
	)
}

// stringForFieldReference returns the string representation of the
// given field reference.
func stringForFieldReference(fieldReference *ast.FieldReferenceNode) string {
	var result string
	if fieldReference.Open != nil {
		result += "("
	}
	result += string(fieldReference.Name.AsIdentifier())
	if fieldReference.Close != nil {
		result += ")"
	}
	return result
}
