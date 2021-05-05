package ast

import (
	"fmt"
	"sort"
)

// FileInfo contains information about the contents of a source file, including
// details about comments and tokens. A lexer accumulates these details as it
// scans the file contents. This allows efficient representation of things like
// source positions.
type FileInfo struct {
	// The name of the source file.
	name string
	// The raw contents of the source file.
	data []byte
	// The offsets for each line in the file. The value is the zero-based byte
	// offset for a given line. The line is given by its index. So the value at
	// index 0 is the offset for the first line (which is always zero). The
	// value at index 1 is the offset at which the second line begins. Etc.
	lines []int
	// The info for every comment in the file. This is empty if the file has no
	// comments. The first entry corresponds to the first comment in the file,
	// and so on.
	comments []commentInfo
	// The info for every token in the file. The last item in the slice
	// corresponds to the EOF, so every file (even an empty one) has at least
	// one element. This includes all terminal symbols in the AST as well as
	// all comments.
	tokens []tokenInfo
}

type commentInfo struct {
	// the item at this index in the associated FileInfo's tokens slice
	// indicates the position and size of the comment.
	index int
	// the item at this index in the associated FileInfo's tokens slice
	// indicates the token to which this comment is attributed.
	attributedToken int
}

type tokenInfo struct {
	// the offset into the file of the first character of a token.
	offset int
	// the length of the token
	length int
}

// NewFileInfo creates a new instance for the given file.
func NewFileInfo(filename string, contents []byte) *FileInfo {
	return &FileInfo{
		name: filename,
		data: contents,
		lines: []int{0},
	}
}

// AddLine adds the offset of a new line in the file. As the lexer encounters
// newlines, it should call this method with the offset of the newline character.
func (f *FileInfo) AddLine(offset int) {
	if offset < 0 {
		panic(fmt.Sprintf("invalid offset: %d must not be negative", offset))
	}
	if offset > len(f.data) {
		panic(fmt.Sprintf("invalid offset: %d is greater than file size %d", offset, len(f.data)))
	}

	if len(f.lines) > 0 {
		lastOffset := f.lines[len(f.lines)-1]
		if offset <= lastOffset {
			panic(fmt.Sprintf("invalid offset: %d is not greater than previously observed line offset %d", offset, lastOffset))
		}
	}

	f.lines = append(f.lines, offset)
}

// AddToken adds info about a token at the given location to this file. It
// returns a value that allows access to all of the token's details.
func (f *FileInfo) AddToken(offset, length int) TokenInfo_ {
	if offset < 0 {
		panic(fmt.Sprintf("invalid offset: %d must not be negative", offset))
	}
	if length < 0 {
		panic(fmt.Sprintf("invalid length: %d must not be negative", length))
	}
	if offset + offset > len(f.data) {
		panic(fmt.Sprintf("invalid offset+length: %d is greater than file size %d", offset+length, len(f.data)))
	}

	if len(f.tokens) > 0 {
		lastToken := f.tokens[len(f.tokens)-1]
		lastEnd := lastToken.offset + lastToken.length - 1
		if offset <= lastEnd {
			panic(fmt.Sprintf("invalid offset: %d is not greater than previously observed token end %d", offset, lastEnd))
		}
	}

	f.tokens = append(f.tokens, tokenInfo{offset: offset, length: length})
	return TokenInfo_{
		fileInfo: f,
		index:    len(f.tokens) - 1,
	}
}

// AddComment adds info about a comment to this file. Comments must first be
// added as tokens via f.AddToken(). The given comment argument is the TokenInfo
// from that step. The given attributedTo argument indicates another token in the
// file with which the comment is associated. If comment's offset is before that
// of attributedTo, then this is a leading comment. Otherwise, it is a trailing
// comment.
func (f *FileInfo) AddComment(comment, attributedTo TokenInfo_) Comment_ {
	if comment.fileInfo != f || attributedTo.fileInfo != f {
		panic(fmt.Sprintf("cannot add comment using token from different *FileInfo"))
	}

	if len(f.comments) > 0 {
		lastComment := f.comments[len(f.comments)-1]
		if comment.index <= lastComment.index {
			panic(fmt.Sprintf("invalid index: %d is not greater than previously observed comment index %d", comment.index, lastComment.index))

		}
		if attributedTo.index < lastComment.attributedToken {
			panic(fmt.Sprintf("invalid attribution: %d is not greater than previously observed comment attribution index %d", attributedTo.index, lastComment.attributedToken))
		}
	}

	f.comments = append(f.comments, commentInfo{index: comment.index, attributedToken: attributedTo.index})
	return Comment_{
		fileInfo: f,
		index:    len(f.comments) - 1,
	}
}

func (f *FileInfo) pos(offset int) SourcePos {
	lineNumber := sort.Search(len(f.lines), func(n int) bool {
		return f.lines[n] > offset
	})

	// If it weren't for tabs, we could trivially compute the column
	// just based on offset and the starting offset of lineNumber :(
	// Wish this were more efficient... that would require also storing
	// computed line+column information, which would triple the size of
	// f's tokens slice...
	col := 0
	for i := f.lines[lineNumber-1]; i < offset; i++ {
		if f.data[i] == '\t' {
			nextTabStop := 8 - (col % 8)
			col += nextTabStop
		} else {
			col++
		}
	}

	return SourcePos{
		Filename: f.name,
		Offset:   offset,
		Line:     lineNumber,
		Col:      col + 1,
	}
}

// TokenInfo represents the details for a single token in a source file. A token
// is either a comment or a terminal symbol. Tokens corresponding to comments
// will have no other comments attributed to them (e.g. LeadingComments() and
// TrailingComments() will return empty values).
type TokenInfo_ struct {
	fileInfo *FileInfo
	index    int
}

func (t *TokenInfo_) Start() SourcePos {
	tok := t.fileInfo.tokens[t.index]
	return t.fileInfo.pos(tok.offset)
}

func (t *TokenInfo_) End() SourcePos {
	tok := t.fileInfo.tokens[t.index]
	return t.fileInfo.pos(tok.offset + tok.length - 1)
}

func (t *TokenInfo_) LeadingWhitespace() string {
	tok := t.fileInfo.tokens[t.index]
	var prevEnd int
	if t.index > 0 {
		prevTok := t.fileInfo.tokens[t.index-1]
		prevEnd = prevTok.offset + prevTok.length
	}
	return string(t.fileInfo.data[prevEnd:tok.offset])
}

func (t *TokenInfo_) RawText() string {
	tok := t.fileInfo.tokens[t.index]
	return string(t.fileInfo.data[tok.offset:tok.offset+tok.length])
}

func (t *TokenInfo_) LeadingComments() Comments {
	start := sort.Search(len(t.fileInfo.comments), func(n int) bool {
		return t.fileInfo.comments[n].attributedToken >= t.index
	})

	if start == len(t.fileInfo.comments) || t.fileInfo.comments[start].attributedToken != t.index {
		// no comments associated with this token
		return Comments{}
	}

	tokOffset := t.fileInfo.tokens[t.index].offset
	numComments := 0
	for i := start; i < len(t.fileInfo.comments); i++ {
		comment := t.fileInfo.comments[i]
		if comment.attributedToken == t.index &&
			t.fileInfo.tokens[comment.index].offset < tokOffset {
			numComments++
		} else {
			break
		}
	}

	return Comments{
		fileInfo: t.fileInfo,
		first:    start,
		num:      numComments,
	}
}

func (t *TokenInfo_) TrailingComments() Comments {
	tokOffset := t.fileInfo.tokens[t.index].offset
	start := sort.Search(len(t.fileInfo.comments), func(n int) bool {
		comment := t.fileInfo.comments[n]
		return comment.attributedToken >= t.index &&
			t.fileInfo.tokens[comment.index].offset > tokOffset
	})

	if start == len(t.fileInfo.comments) || t.fileInfo.comments[start].attributedToken != t.index {
		// no comments associated with this token
		return Comments{}
	}

	numComments := 0
	for i := start; i < len(t.fileInfo.comments); i++ {
		comment := t.fileInfo.comments[i]
		if comment.attributedToken == t.index {
			numComments++
		} else {
			break
		}
	}

	return Comments{
		fileInfo: t.fileInfo,
		first:    start,
		num:      numComments,
	}
}

// Comments represents a range of comments.
type Comments struct {
	fileInfo   *FileInfo
	first, num int
}

func (c Comments) Len() int {
	return c.num
}

func (c Comments) Index(i int) Comment_ {
	if i < 0 || i >= c.num {
		panic(fmt.Sprintf("index %d out of range (len = %d)", i, c.num))
	}
	return Comment_{
		fileInfo: c.fileInfo,
		index:    c.first + i,
	}
}

// Comment_ represents a single comment in a source file. It indicates
// the position of the comment and its contents.
type Comment_ struct {
	fileInfo *FileInfo
	index    int
}

func (c *Comment_) Start() SourcePos {
	comment := c.fileInfo.comments[c.index]
	tok := c.fileInfo.tokens[comment.index]
	return c.fileInfo.pos(tok.offset)
}

func (c *Comment_) End() SourcePos {
	comment := c.fileInfo.comments[c.index]
	tok := c.fileInfo.tokens[comment.index]
	return c.fileInfo.pos(tok.offset + tok.length - 1)
}

func (c *Comment_) LeadingWhitespace() string {
	comment := c.fileInfo.comments[c.index]
	tok := c.fileInfo.tokens[comment.index]
	var prevEnd int
	if comment.index > 0 {
		prevTok := c.fileInfo.tokens[comment.index-1]
		prevEnd = prevTok.offset + prevTok.length
	}
	return string(c.fileInfo.data[prevEnd:tok.offset])
}

func (c *Comment_) RawText() string {
	comment := c.fileInfo.comments[c.index]
	tok := c.fileInfo.tokens[comment.index]
	return string(c.fileInfo.data[tok.offset:tok.offset+tok.length])
}
