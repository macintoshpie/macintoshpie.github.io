package main

import (
	"fmt"
	"path/filepath"

	dev "github.com/macintoshpie/listwebsite/dev"
)

const siteData = "me.txt"
const outDir = "build"

var out = filepath.Join(outDir, "index.html")

const siteTemplate = "me.tmpl.html"

const debug = false

func main() {
	dev.BuildHTML(siteData, siteTemplate)
	dev.BuildGemfiles(siteData)
	fmt.Println("Rebuilt site")
}
