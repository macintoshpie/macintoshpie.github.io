package dev

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/macintoshpie/listwebsite/highlighter"
	parser "github.com/macintoshpie/listwebsite/parsers"
	"github.com/macintoshpie/listwebsite/walkers"
)

const outDir = "build"

var out = filepath.Join(outDir, "index.html")

const debug = false

var templateData = struct {
	Me   string
	Date string
}{
	Me:   "",
	Date: time.Now().Format("2006-01-02"),
}

func BuildHTML(siteDataFile, siteTemplateFile string) {
	siteTxt, err := os.Open(siteDataFile)
	if err != nil {
		panic(err)
	}
	defer siteTxt.Close()

	root := parser.Parse(siteTxt)

	dataOutput, err := os.CreateTemp(os.TempDir(), "listwebsite-*.html")
	if err != nil {
		panic(err)
	}
	defer dataOutput.Close()

	walker := walkers.NewWalker()
	// use the xml package to construct the html
	encoder := xml.NewEncoder(dataOutput)
	if debug {
		encoder.Indent("", "  ")
	}

	walker.AddEventListener(parser.RenderableBlockTypes[:], walkers.ListenerConfig{
		OnEnter: func(node *parser.Node) {
			attrs := []xml.Attr{
				{Name: xml.Name{Local: "name"}, Value: node.Parent.ID},
			}
			encodeStartTag(encoder, "details", attrs...)
			attrs = nil
			if node.Content == "css" {
				attrs = []xml.Attr{{Name: xml.Name{Local: "id"}, Value: "sillyCss"}}
			}
			encodeStartTag(encoder, "summary", attrs...)
			encoder.EncodeToken(xml.CharData(getNodeSummary(node)))
			encodeEndTag(encoder, "summary")
			encodeStartTag(encoder, "p")
		},
		OnExit: func(node *parser.Node) {
			encodeEndTag(encoder, "p")
			encodeEndTag(encoder, "details")
		},
	})

	walker.AddEventListener([]parser.BlockType{parser.BlockLink}, walkers.ListenerConfig{
		OnEnter: func(node *parser.Node) {
			if isProbablyImage(node.Content) {
				attrs := []xml.Attr{
					{Name: xml.Name{Local: "src"}, Value: node.Content},
					{Name: xml.Name{Local: "alt"}, Value: node.Content},
					{Name: xml.Name{Local: "loading"}, Value: "lazy"},
				}
				encodeStartTag(encoder, "img", attrs...)
				encodeEndTag(encoder, "img")
			} else if isProbablyYouTube(node.Content) {
				attrs := []xml.Attr{
					{Name: xml.Name{Local: "src"}, Value: node.Content},
					{Name: xml.Name{Local: "loading"}, Value: "lazy"},
					{Name: xml.Name{Local: "frameborder"}, Value: "0"},
					{Name: xml.Name{Local: "allow"}, Value: "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"},
					{Name: xml.Name{Local: "allowfullscreen"}, Value: "true"},
				}
				encodeStartTag(encoder, "iframe", attrs...)
				encodeEndTag(encoder, "iframe")
			} else {
				encodeStartTag(encoder, "a", xml.Attr{Name: xml.Name{Local: "href"}, Value: node.Content})
				encoder.EncodeToken(xml.CharData(node.Content))
				encodeEndTag(encoder, "a")
			}
		},
		OnExit: func(node *parser.Node) {
		},
	})

	walker.AddEventListener([]parser.BlockType{parser.BlockPreformatted}, walkers.ListenerConfig{
		OnEnter: func(node *parser.Node) {
			codeLines := highlighter.ParseCode(node.Content)
			encodeStartTag(encoder, "div", xml.Attr{Name: xml.Name{Local: "class"}, Value: "code-block"})
			for _, line := range codeLines {
				encodeStartTag(encoder, "div", xml.Attr{Name: xml.Name{Local: "class"}, Value: "code-line"})
				for _, span := range line {
					encodeStartTag(encoder, "span", xml.Attr{Name: xml.Name{Local: "class"}, Value: span.Kind})
					encoder.EncodeToken(xml.CharData(span.Value))
					encodeEndTag(encoder, "span")
				}
				encodeEndTag(encoder, "div")
			}
			encodeEndTag(encoder, "div")
		},
		OnExit: func(node *parser.Node) {
		},
	})

	walker.Walk(root)
	err = encoder.Flush()
	if err != nil {
		panic(err)
	}
	err = encoder.Close()
	if err != nil {
		panic(err)
	}

	// now drop the output into the template
	templateFile, err := os.Open(siteTemplateFile)
	if err != nil {
		panic(err)
	}
	defer templateFile.Close()

	// use the text/template package to render the template to avoid escaping the html
	templateRoot, err := template.ParseFiles(siteTemplateFile)
	if err != nil {
		panic(err)
	}

	outFile, err := os.Create(out)
	if err != nil {
		panic(err)
	}

	dataBytes, err := os.ReadFile(dataOutput.Name())
	if err != nil {
		panic(err)
	}

	templateData.Me = string(dataBytes)
	templateRoot.Execute(outFile, templateData)
}

type GeminiWalkerDir struct {
	node      *parser.Node
	path      string
	indexFile *os.File
}

type GeminiWalkerCtx struct {
	root          *parser.Node
	dirBlockTypes []parser.BlockType
	// Maps node ID to the directory data
	dirData map[string]*GeminiWalkerDir
}

func BuildGemfiles(siteDataFile string) {
	siteTxt, err := os.Open(siteDataFile)
	if err != nil {
		panic(err)
	}
	defer siteTxt.Close()

	root := parser.Parse(siteTxt)

	geminiCtx := GeminiWalkerCtx{
		root:          root,
		dirBlockTypes: []parser.BlockType{parser.BlockPage},
		dirData:       map[string]*GeminiWalkerDir{},
	}
	geminiCtx.init()

	walker := walkers.NewWalker()
	walker.AddEventListener(geminiCtx.dirBlockTypes, walkers.ListenerConfig{
		OnEnter: func(node *parser.Node) {
			// create a directory for the page under the parent page (or root)
			parentData, err := geminiCtx.findDirParentData(node)
			if err != nil {
				panic(err)
			}

			dirPath := filepath.Join(parentData.path, slugify(node.Content))
			err = os.MkdirAll(filepath.Join(outDir, dirPath), 0755)
			if err != nil {
				panic(err)
			}

			// create index file
			indexFile, err := os.Create(filepath.Join(outDir, dirPath, "index.gmi"))
			if err != nil {
				panic(err)
			}
			indexFile.WriteString("# " + node.Content + "\n\n")
			geminiCtx.dirData[node.ID] = &GeminiWalkerDir{
				node:      node,
				path:      dirPath,
				indexFile: indexFile,
			}

			// add a link to this page from parent
			parentData.indexFile.WriteString("=> " + dirPath + "\n")

			// add a link back to the parent
			indexFile.WriteString("=> " + parentData.path + "\n")
		},
		OnExit: func(node *parser.Node) {
			// we can close the file now that all children are done
			if data, ok := geminiCtx.dirData[node.ID]; ok {
				data.indexFile.Close()
			}
		},
	})

	fileContentBlockTypes := []parser.BlockType{}
	for _, blockType := range parser.RenderableBlockTypes {
		if !slices.Contains(geminiCtx.dirBlockTypes, blockType) {
			fileContentBlockTypes = append(fileContentBlockTypes, blockType)
		}
	}
	walker.AddEventListener(fileContentBlockTypes, walkers.ListenerConfig{
		OnEnter: func(node *parser.Node) {
			// write the content to the index file
			parentData, err := geminiCtx.findDirParentData(node)
			if err != nil {
				panic(err)
			}

			switch node.BlockType {
			case parser.BlockText:
				thisNodeDepth := node.Depth - parentData.node.Depth
				parentData.indexFile.WriteString("* " + strings.Repeat(".", thisNodeDepth) + " " + node.Content + "\n")
			case parser.BlockLink:
				parentData.indexFile.WriteString("=> " + node.Content + "\n")
			case parser.BlockPreformatted:
				parentData.indexFile.WriteString("```" + node.Content + "\n```" + "\n")
			case parser.BlockHeader:
				parentData.indexFile.WriteString("# " + node.Content + "\n")
			case parser.BlockQuote:
				parentData.indexFile.WriteString("> " + node.Content + "\n")
			default:
				panic(fmt.Sprintf("unhandled block type: %s", node.BlockType.Name()))
			}
		},
		OnExit: func(node *parser.Node) {
		},
	})

	walker.Walk(root)
}

// init initializes the GeminiWalkerCtx and must be called before walking the tree
func (g *GeminiWalkerCtx) init() {
	// create root
	rootIndex := filepath.Join(outDir, "index.gmi")
	rootFile, err := os.Create(rootIndex)
	if err != nil {
		panic(err)
	}

	data := &GeminiWalkerDir{
		node:      g.root,
		path:      "/",
		indexFile: rootFile,
	}

	g.dirData[g.root.ID] = data
}

func (g *GeminiWalkerCtx) findDirParent(node *parser.Node) *parser.Node {
	parentNode := node.Parent
	for parentNode != nil &&
		!slices.Contains(g.dirBlockTypes, parentNode.BlockType) &&
		// root node is implicitly a dir block type
		parentNode.BlockType != parser.BlockRoot {
		parentNode = parentNode.Parent
	}

	return parentNode
}

func (g *GeminiWalkerCtx) findDirParentData(node *parser.Node) (*GeminiWalkerDir, error) {
	parentNode := g.findDirParent(node)
	if parentNode == nil {
		return nil, fmt.Errorf("no directory data found for node: %s (%s)", node.Content, node.BlockType.Name())
	}

	if data, ok := g.dirData[parentNode.ID]; ok {
		return data, nil
	}

	return nil, fmt.Errorf("no directory data found for node: %s (%s)", node.Content, node.BlockType.Name())
}

// Returns a valid slug usable for url and file directory names
func slugify(s string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	return strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func encodeStartTag(e *xml.Encoder, name string, attrs ...xml.Attr) error {
	return e.EncodeToken(xml.StartElement{Name: xml.Name{Local: name}, Attr: attrs})
}

func encodeEndTag(e *xml.Encoder, name string) error {
	return e.EncodeToken(xml.EndElement{Name: xml.Name{Local: name}})
}

func isProbablyImage(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".gif")
}

func isProbablyYouTube(s string) bool {
	return strings.Contains(s, "youtube.com/embed")
}

func getNodeSummary(node *parser.Node) string {
	if node.BlockType == parser.BlockPreformatted {
		// return first 30 characters
		if len(node.Content) > 30 {
			return fmt.Sprintf("`code` %s...", node.Content[:30])
		}

		return node.Content
	}

	return node.Content
}
