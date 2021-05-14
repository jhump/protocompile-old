package parser

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhump/protocompile/reporter"
)

func TestBasicSuccess(t *testing.T) {
	r := readerForTestdata(t, "largeproto.proto")
	handler := reporter.NewHandler(nil)

	fileNode, err := Parse("largeproto.proto", r, handler)
	require.NoError(t, err)

	result, err := ResultFromAST(fileNode, true, handler)
	require.NoError(t, err)
	require.NoError(t, handler.Error())

	assert.Equal(t, "proto3", result.AST().Syntax.Syntax.AsString())
}

func BenchmarkBasicSuccess(b *testing.B) {
	r := readerForTestdata(b, "largeproto.proto")
	bs, err := io.ReadAll(r)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.ReportAllocs()
		byteReader := bytes.NewReader(bs)
		handler := reporter.NewHandler(nil)

		fileNode, err := Parse("largeproto.proto", byteReader, handler)
		require.NoError(b, err)

		result, err := ResultFromAST(fileNode, true, handler)
		require.NoError(b, err)
		require.NoError(b, handler.Error())

		assert.Equal(b, "proto3", result.AST().Syntax.Syntax.AsString())
	}
}

func readerForTestdata(t testing.TB, filename string) io.Reader {
	file, err := os.Open(filepath.Join("testdata", filename))
	require.NoError(t, err)

	return file
}
