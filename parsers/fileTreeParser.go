package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func ParseFileTree(codeDir string, fileTypes []string, additionalFilePaths []string) *Node {
	root := &Node{ID: "root", Depth: -1, Content: "root", Parent: nil, Children: []*Node{}}
	fmt.Println(root)
	codeFiles := []string{}
	err := filepath.Walk(codeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && slices.Contains(fileTypes, filepath.Ext(path)) {
			completeNode(path, root)
			codeFiles = append(codeFiles, path)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	for _, path := range additionalFilePaths {
		completeNode(path, root)
	}

	return root
}

func completeNode(path string, root *Node) {
	fmt.Println("Doing path", path)
	parts := strings.Split(path, "/")

	currentNode := root
	currentDepth := 0
	for _, part := range parts {
		for _, child := range currentNode.Children {
			if child.Content == part {
				currentNode = child
				currentDepth++
				break
			}
		}

		// doesn't already exist, create it
		fmt.Println("Creating", part)
		newNode := &Node{
			Depth:     currentDepth,
			Content:   part,
			BlockType: BlockText,
			Parent:    currentNode,
			Children:  []*Node{},
			ID:        "",
		}
		currentNode.Children = append(currentNode.Children, newNode)

		// if this is the last part in path, add the code
		if part == parts[len(parts)-1] {
			addCodeNode(newNode, path, currentDepth)
		}

		currentNode = currentNode.Children[len(currentNode.Children)-1]
		currentDepth++
	}
}

func addCodeNode(parent *Node, path string, currentDepth int) {
	// read the file and parse it
	code, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	codeStr := string(code)
	parent.Children = append(parent.Children, &Node{
		Depth:     currentDepth + 1,
		Content:   codeStr,
		BlockType: BlockPreformatted,
		Parent:    parent,
		Children:  []*Node{},
		ID:        "",
	})
}
