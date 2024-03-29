package protocompile

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protocompile/ast"
)

// Resolver is used by the compiler to resolve a proto source file name
// into some unit that is usable by the compiler. The result could be source
// for a proto file or it could be an already-parsed AST or descriptor.
type Resolver interface {
	// FindFileByPath searches for information for the given file path. If no
	// result is available, it should return a non-nil error, such as
	// protoregistry.NotFound.
	FindFileByPath(path string) (SearchResult, error)
}

// SearchResult represents information about a proto source file. Only one of
// the various fields must be set, based on what is available for a file. If
// multiple fields are set, the compiler prefers them in opposite order listed:
// so it uses a descriptor if present and only falls back to source if nothing
// else if available.
type SearchResult struct {
	// Represents source code for the file. This should be nil if source code
	// is not available. If no field below is set, then the compiler will parse
	// the source code into an AST.
	Source io.Reader
	// Represents the abstract syntax tree for the file. If no field below is
	// set, then the compiler will convert the AST into a descriptor proto.
	AST *ast.FileNode
	// A descriptor proto that represents the file. If the field below is not
	// set, then the compiler will link this proto with its dependencies to
	// produce a linked descriptor.
	Proto *descriptorpb.FileDescriptorProto
	// A fully linked descriptor that represents the file. If this field is set,
	// then the compiler has no additional work to do for this file as it is
	// already compiled.
	Desc protoreflect.FileDescriptor
}

// ResolverFunc is a simple function type that implements Resolver.
type ResolverFunc func(string) (SearchResult, error)

var _ Resolver = ResolverFunc(nil)

func (f ResolverFunc) FindFileByPath(path string) (SearchResult, error) {
	return f(path)
}

// CompositeResolver is a slice of resolvers, which are consulted in order
// until one can supply a result. If none of the constituent resolvers can
// supply a result, the error returned by the first resolver is returned. If
// the slice of resolvers is empty, all operations return
// protoregistry.NotFound.
type CompositeResolver []Resolver

var _ Resolver = CompositeResolver(nil)

func (f CompositeResolver) FindFileByPath(path string) (SearchResult, error) {
	if len(f) == 0 {
		return SearchResult{}, protoregistry.NotFound
	}
	var firstErr error
	for _, res := range f {
		r, err := res.FindFileByPath(path)
		if err == nil {
			return r, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return SearchResult{}, firstErr
}

// SourceResolver can resolve file names by returning source code. It uses
// an optional list of import paths to search. By default, it searches the
// file system.
type SourceResolver struct {
	// Optional list of import paths. If present and not empty, then all
	// file paths to find are assumed to be relative to one of these paths.
	// If nil or empty, all file paths to find are assumed to be relative to
	// the current working directory.
	ImportPaths []string
	// Optional function for returning a file's contents. If nil, then
	// os.Open is used to open files on the file system.
	Accessor func(path string) (io.ReadCloser, error)
}

var _ Resolver = (*SourceResolver)(nil)

func (r *SourceResolver) FindFileByPath(path string) (SearchResult, error) {
	if len(r.ImportPaths) == 0 {
		reader, err := r.accessFile(path)
		if err != nil {
			return SearchResult{}, err
		}
		return SearchResult{Source: reader}, nil
	}

	var e error
	for _, importPath := range r.ImportPaths {
		reader, err := r.accessFile(filepath.Join(importPath, path))
		if err != nil {
			if os.IsNotExist(err) {
				e = err
				continue
			}
			return SearchResult{}, err
		}
		return SearchResult{Source: reader}, nil
	}
	return SearchResult{}, e
}

func (r *SourceResolver) accessFile(path string) (io.ReadCloser, error) {
	if r.Accessor != nil {
		return r.Accessor(path)
	}
	return os.Open(path)
}

// SourceAccessorFromMap returns a function that can be used as the Accessor
// field of a SourceResolver that uses the given map to load source. The map
// keys are file names and the values are the corresponding file contents.
func SourceAccessorFromMap(srcs map[string]string) func(string) (io.ReadCloser, error) {
	return func(path string) (io.ReadCloser, error) {
		src, ok := srcs[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		return io.NopCloser(strings.NewReader(src)), nil
	}
}

// WithStandardImports returns a new resolver that knows about the same standard
// imports that are included with protoc.
func WithStandardImports(r Resolver) Resolver {
	return ResolverFunc(func(name string) (SearchResult, error) {
		res, err := r.FindFileByPath(name)
		if err != nil {
			// error from given resolver? see if it's a known standard file
			if d, ok := standardImports[name]; ok {
				return SearchResult{Desc: d}, nil
			}
		}
		return res, err
	})
}
