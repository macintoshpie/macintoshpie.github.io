package main

import (
	"encoding/xml"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/macintoshpie/listwebsite/dev"
	"github.com/macintoshpie/listwebsite/highlighter"
	"github.com/macintoshpie/listwebsite/monitors"
	parser "github.com/macintoshpie/listwebsite/parsers"
	"github.com/macintoshpie/listwebsite/walkers"
)

const siteData = "me.txt"
const out = "build/index.html"
const siteTemplate = "me.tmpl.html"

const debug = false

var templateData = struct {
	Me   string
	Date string
}{
	Me:   "",
	Date: time.Now().Format("2006-01-02"),
}

func main() {
	monitor, err := monitors.NewFileMonitor([]string{siteData, siteTemplate})

	if err != nil {
		panic(err)
	}

	done := make(chan bool)

	go func() {
		for file := range monitor.Changed {
			println("File changed:", file)
			buildHTML(siteData, siteTemplate)
		}
	}()

	go func() {
		dev.ServeDirectory("build")
	}()

	go func() {
		dev.GeminiServeDirectory("build")
	}()

	<-done
}

func buildHTML(siteDataFile, siteTemplateFile string) {
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
			encoder.EncodeToken(xml.CharData(node.Content))
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
