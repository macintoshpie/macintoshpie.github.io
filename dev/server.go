package dev

import (
	"fmt"
	"net/http"
)

const addr = ":8080"

func ServeDirectory(dir string) {
	// serve files from the directory
	http.Handle("/", http.FileServer(http.Dir(dir)))

	fmt.Println("Serving on", addr)
	// start the server
	http.ListenAndServe(addr, nil)
}
