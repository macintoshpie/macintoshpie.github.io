package renderers

import (
	"io"
	"strings"

	parser "github.com/macintoshpie/listwebsite/parsers"
)

type ListRenderer struct {
	indentation  string
	delim        string
	tokenPadding string
}

func NewListRenderer() *ListRenderer {
	return &ListRenderer{
		indentation:  " ",
		delim:        "-",
		tokenPadding: " ",
	}
}

func (lr *ListRenderer) WithIndentation(indentation string) *ListRenderer {
	lr.indentation = indentation
	return lr
}

func (lr *ListRenderer) WithDelim(delim string) *ListRenderer {
	lr.delim = delim
	return lr
}

func (lr *ListRenderer) WithTokenPadding(tokenPadding string) *ListRenderer {
	lr.tokenPadding = tokenPadding
	return lr
}

func (lr *ListRenderer) Render(node *parser.Node, o io.Writer) {
	if node.BlockType != parser.BlockRoot {
		o.Write([]byte(strings.Repeat(lr.indentation, node.Depth)))
		o.Write([]byte(lr.delim))
		o.Write([]byte(lr.tokenPadding))
		if node.BlockType.Token() != "" {
			o.Write([]byte(node.BlockType.Token()))
			o.Write([]byte(lr.tokenPadding))
		}
		// escape the content
		content := strings.ReplaceAll(node.Content, "```", "\\`\\`\\`")
		o.Write([]byte(content + "\n"))

		if node.BlockType == parser.BlockPreformatted {
			o.Write([]byte("```\n"))
		}
	}

	for _, child := range node.Children {
		lr.Render(child, o)
	}
}
