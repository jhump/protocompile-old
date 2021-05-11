package parser

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jhump/protocompile/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicSuccess(t *testing.T) {
	r := readerForTestdata(t, "largeproto.proto")
	handler := reporter.NewHandler(&testReporter{t: t})

	fileNode, err := Parse("largeproto.proto", r, handler)
	require.NoError(t, err)

	assert.Equal(t, "proto3", fileNode.Syntax.Syntax.AsString())
}

func BenchmarkBasicSuccess(b *testing.B) {
	r := readerForTestdata(b, "largeproto.proto")
	bs, err := io.ReadAll(r)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.ReportAllocs()
		byteReader := bytes.NewReader(bs)
		handler := reporter.NewHandler(&testReporter{t: b})

		fileNode, err := Parse("largeproto.proto", byteReader, handler)
		require.NoError(b, err)

		assert.Equal(b, "proto3", fileNode.Syntax.Syntax.AsString())
	}
}

func readerForTestdata(t testing.TB, filename string) io.Reader {
	file, err := os.Open(filepath.Join("testdata", filename))
	require.NoError(t, err)

	return file
}

type testReporter struct {
	t testing.TB
}

func (r *testReporter) Error(errWithPos reporter.ErrorWithPos) error {
	r.t.Logf("Parser error: %s", errWithPos.Error())
	r.t.FailNow()
	return errWithPos
}

func (r *testReporter) Warning(errWithPos reporter.ErrorWithPos) {
	r.t.Logf("Parser warning: %s", errWithPos.Error())
	r.t.Fail()
}
