package renderers

import (
	"encoding/xml"

	parser "github.com/macintoshpie/listwebsite/parsers"
)

func NewUlRenderer(node *parser.Node) *XMLRenderer {
	return &XMLRenderer{
		root: newXMLRendererNode(node, ulEncoder),
	}
}

func NewDetailsRenderer(node *parser.Node) *XMLRenderer {
	return &XMLRenderer{
		root: newXMLRendererNode(node, detailsEncoder),
	}
}

func detailsEncoder(node *xmlRendererNode, e *xml.Encoder, start xml.StartElement) error {
	if node.BlockType != parser.BlockRoot {
		// set the name of the custom xml element
		start.Name.Local = "details"
		err := e.EncodeToken(start)
		if err != nil {
			return err
		}

		defer func() {
			// close the custom xml element
			err := e.EncodeToken(xml.EndElement{Name: start.Name})
			if err != nil {
				panic(err)
			}
		}()

		// encode summary
		err = e.EncodeElement(node.Content, xml.StartElement{Name: xml.Name{Local: "summary"}})
		if err != nil {
			return err
		}
	}

	// encode children, if any
	for _, child := range node.Children {
		// Recursively call MarshalXML on each child
		err := e.EncodeElement(child, xml.StartElement{Name: xml.Name{Local: "placeholder"}})
		if err != nil {
			return err
		}
	}

	return nil
}

func ulEncoder(node *xmlRendererNode, e *xml.Encoder, start xml.StartElement) error {
	if node.BlockType == parser.BlockRoot {
		// set the name of the custom xml element
		start.Name.Local = "ul"
		err := e.EncodeToken(start)
		if err != nil {
			return err
		}

		// encode children, if any
		for _, child := range node.Children {
			// Recursively call MarshalXML on each child
			err := e.EncodeElement(child, xml.StartElement{Name: xml.Name{Local: "placeholder"}})
			if err != nil {
				return err
			}
		}

		err = e.EncodeToken(xml.EndElement{Name: start.Name})
		return err
	} else {
		// set the name of the custom xml element
		start.Name.Local = "li"
		err := e.EncodeToken(start)
		if err != nil {
			return err
		}

		err = e.EncodeToken(xml.CharData(node.Content))
		if err != nil {
			return err
		}

		if len(node.Children) > 0 {
			err = e.EncodeToken(xml.StartElement{Name: xml.Name{Local: "ul"}})
			if err != nil {
				return err
			}

			for _, child := range node.Children {
				// Recursively call MarshalXML on each child
				err := e.EncodeElement(child, xml.StartElement{Name: xml.Name{Local: "placeholder"}})
				if err != nil {
					return err
				}
			}

			err = e.EncodeToken(xml.EndElement{Name: xml.Name{Local: "ul"}})
			if err != nil {
				return err
			}
		}

		err = e.EncodeToken(xml.EndElement{Name: start.Name})
		return err
	}
}
