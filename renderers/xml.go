package renderers

import (
	"encoding/xml"
	"io"

	parser "github.com/macintoshpie/listwebsite/parsers"
)

// NewXmlRenderer creates a new XMLRenderer with the given parser.Node as the root
func NewXmlRenderer(node *parser.Node) *XMLRenderer {
	return &XMLRenderer{
		root: newXMLRendererNode(node, simpleXmlEncoder),
	}
}

// SetXmlEncoder sets the encoder function for the given node and all of its children
func SetXmlEncoder(node *xmlRendererNode, encoderFunc EncoderFunc) {
	node.encoderFunc = encoderFunc

	for _, child := range node.Children {
		SetXmlEncoder(child, encoderFunc)
	}
}

type XMLRenderer struct {
	root *xmlRendererNode
}

func (xr *XMLRenderer) Render(o io.Writer) {
	xmlResult, err := xml.MarshalIndent(xr.root, "", "  ")
	if err != nil {
		panic(err)
	}

	o.Write(xmlResult)
}

type EncoderFunc func(node *xmlRendererNode, e *xml.Encoder, start xml.StartElement) error

// xmlRendererNode is a struct that represents a node that can be rendered to XML
// I tried using embedding to avoid recreating nodes, but the Renderer would end up using the parser.Node struct's Renderer instead...
type xmlRendererNode struct {
	BlockType   parser.BlockType
	Content     string
	Children    []*xmlRendererNode
	encoderFunc EncoderFunc
}

func (nm *xmlRendererNode) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return nm.encoderFunc(nm, e, start)
}

// newXMLRendererNode recursively creates a new XMLRendererNode tree from a parser.Node tree
func newXMLRendererNode(node *parser.Node, encoderFunc EncoderFunc) *xmlRendererNode {
	nm := &xmlRendererNode{
		BlockType:   node.BlockType,
		Content:     node.Content,
		encoderFunc: encoderFunc,
	}
	for _, child := range node.Children {
		nm.Children = append(nm.Children, newXMLRendererNode(child, encoderFunc))
	}
	return nm
}

func simpleXmlEncoder(node *xmlRendererNode, e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = node.BlockType.Name()
	err := e.EncodeToken(start)
	if err != nil {
		return err
	}

	err = e.EncodeToken(xml.CharData(node.Content))
	if err != nil {
		return err
	}

	for _, child := range node.Children {
		err = e.EncodeElement(child, xml.StartElement{Name: xml.Name{Local: "placeholder"}})
		if err != nil {
			return err
		}
	}

	err = e.EncodeToken(xml.EndElement{Name: start.Name})
	return err
}
