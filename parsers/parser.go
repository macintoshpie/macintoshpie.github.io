package parser

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	BlockRoot = iota
	BlockText
	BlockPreformatted
	BlockHeader
	BlockPage
	BlockLink
	BlockQuote
)

type BlockType int

var AllBlockTypes = [...]BlockType{
	BlockRoot,
	BlockText,
	BlockPreformatted,
	BlockHeader,
	BlockPage,
	BlockLink,
	BlockQuote,
}

var RenderableBlockTypes = [...]BlockType{
	BlockText,
	BlockPreformatted,
	BlockHeader,
	BlockPage,
	BlockLink,
	BlockQuote,
}

type blockSpec struct {
	BlockType BlockType
	Token     string
	Name      string
}

var blockTypeToString = map[BlockType]blockSpec{
	BlockRoot: {
		BlockType: BlockRoot,
		Token:     "__root__",
		Name:      "Root",
	},
	BlockText: {
		BlockType: BlockText,
		Token:     "",
		Name:      "Text",
	},
	BlockPreformatted: {
		BlockType: BlockPreformatted,
		Token:     "```",
		Name:      "Preformatted",
	},
	BlockHeader: {
		BlockType: BlockHeader,
		Token:     "#",
		Name:      "Header",
	},
	BlockPage: {
		BlockType: BlockPage,
		Token:     "$",
		Name:      "Page",
	},
	BlockLink: {
		BlockType: BlockLink,
		Token:     "=>",
		Name:      "Link",
	},
	BlockQuote: {
		BlockType: BlockQuote,
		Token:     ">",
		Name:      "Quote",
	},
}

func (blockType BlockType) Name() string {
	if spec, ok := blockTypeToString[blockType]; ok {
		return spec.Name
	}
	panic("Unknown block type: " + strconv.Itoa(int(blockType)))
}

func (blockType BlockType) Token() string {
	if spec, ok := blockTypeToString[blockType]; ok {
		return spec.Token
	}
	panic("Unknown block type: " + strconv.Itoa(int(blockType)))
}

type Node struct {
	Depth     int
	Content   string
	BlockType BlockType
	Parent    *Node
	Children  []*Node
	ID        string
}

const (
	ParseNormal = iota
	ParsePreFormatted
)

// const fileName = "me.txt"
const fileName = "test.txt"

func Parse(content io.Reader) *Node {
	scanner := bufio.NewScanner(content)

	currentMode := ParseNormal
	rootNode := &Node{ID: "root", Depth: -1, Content: "root", Parent: nil, Children: []*Node{}}
	currentNode := rootNode
	currentLine := 0
	for scanner.Scan() {
		currentLine++
		lineData := scanner.Text()

		switch currentMode {
		case ParseNormal:
			charIndex := 0
			for charIndex < len(lineData) && lineData[charIndex] == ' ' {
				charIndex++
			}
			spaceCount := charIndex

			if charIndex == len(lineData) {
				continue
			}

			if lineData[charIndex] != '-' {
				panicWith(currentNode, "Expected '-' at line "+strconv.Itoa(currentLine)+" char "+strconv.Itoa(charIndex)+" but got "+string(lineData[charIndex]))
			}

			remainingData := strings.TrimLeft(lineData[charIndex+1:], " ")
			blockType, blockContent := getBlock(remainingData)

			// walk up the tree until we find a parent with smaller depth
			parentNode := currentNode
			for parentNode != nil && parentNode.Depth >= spaceCount {
				parentNode = parentNode.Parent
			}

			if parentNode == nil {
				panicWith(rootNode, "No parent found for line "+strconv.Itoa(currentLine))
			}

			newNode := &Node{Depth: spaceCount, Content: blockContent, BlockType: blockType, Parent: currentNode, Children: []*Node{}}
			newNode.updateID()
			newNode.Parent = parentNode
			parentNode.Children = append(parentNode.Children, newNode)
			currentNode = newNode

			if blockType == BlockPreformatted {
				currentMode = ParsePreFormatted
			}

		case ParsePreFormatted:
			if hasPrefix(lineData, "```") {
				currentMode = ParseNormal
				continue
			} else {
				if currentNode.Content == "" {
					currentNode.Content = lineData
				} else {
					currentNode.Content += "\n" + lineData
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return rootNode
}

func (node *Node) String() string {
	return fmt.Sprintf("[%s]%s", node.BlockType.Name(), node.Content)
}

func (node *Node) FindNode(content string) (*Node, error) {
	if node.Content == content {
		return node, nil
	}

	for _, child := range node.Children {
		foundNode, err := child.FindNode(content)
		if err == nil {
			return foundNode, nil
		}
	}

	return nil, fmt.Errorf("Node with content \"%s\" not found", content)
}

func (node *Node) updateID() {
	node.ID = hashString(node.Content)
}

func hasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
	// return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func getBlock(data string) (blockType BlockType, blockContent string) {
	switch {
	case hasPrefix(data, "```"):
		return BlockPreformatted, data[3:]
	case hasPrefix(data, "#"):
		return BlockHeader, strings.TrimLeft(data[1:], " ")
	case hasPrefix(data, "=>"):
		return BlockLink, strings.TrimLeft(data[2:], " ")
	case hasPrefix(data, "$"):
		return BlockPage, strings.TrimLeft(data[1:], " ")
	case hasPrefix(data, ">"):
		return BlockQuote, strings.TrimLeft(data[1:], " ")
	default:
		return BlockText, data
	}
}

func panicWith(lastNode *Node, message string) {
	panic("Last Node: " + lastNode.String() + "\n" + message)
}

func hashString(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes)
}
