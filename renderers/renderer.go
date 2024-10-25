package renderers

import (
	"io"

	parser "github.com/macintoshpie/listwebsite/parsers"
)

type Renderer interface {
	Render(node *parser.Node, o io.Writer)
}
