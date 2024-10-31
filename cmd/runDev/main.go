package main

import (
	"fmt"
	"os"

	"github.com/macintoshpie/listwebsite/dev"
	"github.com/macintoshpie/listwebsite/monitors"
	parser "github.com/macintoshpie/listwebsite/parsers"
	"github.com/macintoshpie/listwebsite/renderers"
)

const siteData = "me.txt"
const outDir = "build"

const siteTemplate = "me.tmpl.html"

func main() {
	// build the site every time the site data changes
	siteMonitor, err := monitors.NewFileMonitor([]string{siteData, siteTemplate})
	if err != nil {
		panic(err)
	}
	done := make(chan bool)
	go func() {
		for x := range siteMonitor.Changed {
			_ = x
			dev.BuildHTML(siteData, siteTemplate)
			dev.BuildGemfiles(siteData)
			fmt.Println("Rebuilt site")
		}
	}()

	// update the site data every time one of the code file changes
	codeMonitor, err := monitors.NewFileMonitor([]string{"cmd/runDev/main.go", "parsers/parser.go", "highlighter/highlighter.go", "parsers/fileTreeParser.go"})
	if err != nil {
		panic(err)
	}
	go func() {
		for x := range codeMonitor.Changed {
			_ = x
			updateSiteDataSourceCode()
			fmt.Println("Updated site data with source code")
		}
	}()

	go func() {
		dev.ServeDirectory(outDir)
	}()

	go func() {
		dev.GeminiServeDirectory(outDir)
	}()

	<-done
}

func changeDepth(node *parser.Node, newDepth int) {
	node.Depth = newDepth
	for _, child := range node.Children {
		changeDepth(child, newDepth+1)
	}
}

func updateSiteDataSourceCode() {
	siteTxt, err := os.Open(siteData)
	if err != nil {
		panic(err)
	}

	root := parser.Parse(siteTxt)
	siteTxt.Close()
	sourceCodeNode, err := root.FindNode("source code")
	if err != nil {
		panic(err)
	}

	listRenderer := renderers.NewListRenderer()

	// render first without the new code to avoid dupes in me.txt when reading below
	sourceCodeNode.Children = []*parser.Node{}
	siteTxt, err = os.Create(siteData)
	if err != nil {
		panic(err)
	}
	listRenderer.Render(root, siteTxt)
	siteTxt.Close()

	daNode := parser.ParseFileTree(".", []string{".go"}, []string{"me.txt", "me.tmpl.html"})
	// since this is getting moved into a subtree, we need to change the depth of the node
	changeDepth(daNode, sourceCodeNode.Depth+1)
	// since parsefiletree returns a root (and we already have one in the tree we're editing) we need to append the children
	sourceCodeNode.Children = append(sourceCodeNode.Children, daNode.Children...)
	for _, child := range daNode.Children {
		child.Parent = sourceCodeNode
	}

	siteTxt, err = os.Create(siteData)
	if err != nil {
		panic(err)
	}
	listRenderer.Render(root, siteTxt)
	siteTxt.Close()
}
