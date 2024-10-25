package dev

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const httpAddr = ":8080"
const geminiAddr = ":8081"
const geminiCertFile = "dev-cert.pem"
const geminiKeyFile = "dev-key.pem"

var geminiLogger = log.New(os.Stdout, "[Gemini Server] ", log.LstdFlags|log.Lshortfile)

func ServeDirectory(dir string) {
	// serve files from the directory
	http.Handle("/", http.FileServer(http.Dir(dir)))

	fullAddr := fmt.Sprintf("http://localhost%s", httpAddr)
	fmt.Println("[HTTP Server] Serving HTTP on", fullAddr)
	// start the server
	http.ListenAndServe(httpAddr, nil)
}

func GeminiServeDirectory(dir string) {

	cert, err := tls.LoadX509KeyPair(geminiCertFile, geminiKeyFile)
	if err != nil {
		panic(err)
	}
	config := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		MaxVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
	}
	ln, err := tls.Listen("tcp", geminiAddr, config)
	if err != nil {
		panic(err)
	}

	fullAddr := fmt.Sprintf("gemini://localhost%s", geminiAddr)
	geminiLogger.Println("Serving Gemini on", fullAddr, "from", dir)
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handleGeminiRequest(conn, dir)
	}
}

const responseInvalidRequest = "59 Invalid request\r\n"

// handleGeminiRequest handles a single Gemini request
// Docs:
// https://geminiprotocol.net/docs/protocol-specification.gmi
// gemini://geminiprotocol.net/docs/protocol-specification.gmi
func handleGeminiRequest(conn net.Conn, dir string) {
	absBaseDir, err := filepath.Abs(dir)

	if err != nil {
		geminiLogger.Println("Error getting absolute path for base directory:", dir)
		conn.Write([]byte(responseInvalidRequest))
		return
	}

	defer conn.Close()

	uriMaxBytes := 1024
	// +2 for \r\n
	req := make([]byte, uriMaxBytes+2)
	bytesRead, err := conn.Read(req)
	if err != nil {
		panic(err)
	}

	uriString := string(req[:bytesRead])
	if uriString[len(uriString)-2:] != "\r\n" {
		geminiLogger.Println("Request too long or missing CRLF:", uriString)
		conn.Write([]byte(responseInvalidRequest))
		return
	}

	uriString = uriString[:len(uriString)-2]

	uriReq, err := url.ParseRequestURI(uriString)
	if err != nil {
		geminiLogger.Println("Invalid URI:", uriString)
		conn.Write([]byte(responseInvalidRequest))
		return
	}

	dirPath := filepath.Join(absBaseDir, uriReq.Path)
	resourcePath, err := filepath.Abs(dirPath)
	if err != nil {
		geminiLogger.Println("Error getting absolute path from requested filepath:", dirPath)
		conn.Write([]byte(responseInvalidRequest))
		return
	}

	// check if the requested file is in the base directory
	// haven't been able to sufficiently test this so it could be insecure
	isServable := strings.HasPrefix(resourcePath, absBaseDir+string(filepath.Separator))
	if !isServable {
		geminiLogger.Println("Requested file is not in the base directory:", resourcePath)
		conn.Write([]byte(responseInvalidRequest))
		return
	}

	response := handleGeminiFileRequest(resourcePath, true)
	geminiLogger.Printf("%s -> %d\n", uriReq.Path, response.statusCode)
	response.Write(conn)
}

type GeminiResponse struct {
	statusCode int
	mimeType   string
	content    string
	errorMsg   string
}

func (r GeminiResponse) Write(w io.Writer) {
	responseType := string(strconv.Itoa(r.statusCode)[0])
	switch responseType {
	case "2":
		w.Write([]byte(fmt.Sprintf("%d %s\r\n", r.statusCode, r.mimeType)))
		w.Write([]byte(r.content))
	case "4", "5":
		w.Write([]byte(fmt.Sprintf("%d %s\r\n", r.statusCode, r.errorMsg)))
	default:
		w.Write([]byte(fmt.Sprintf("%d %s\r\n", 40, "Internal server error")))
	}
}

func handleGeminiFileRequest(path string, allowDir bool) GeminiResponse {
	f, err := os.Stat(path)
	if err != nil {
		return GeminiResponse{
			statusCode: 51,
			errorMsg:   "File not found",
		}
	}

	if f.IsDir() {
		if !allowDir {
			return GeminiResponse{
				statusCode: 51,
				errorMsg:   "File not found",
			}
		}

		// try serving an index.gmi file
		indexPath := filepath.Join(path, "index.gmi")
		return handleGeminiFileRequest(indexPath, false)
	}

	return constructGeminiFileResponse(path)
}

func constructGeminiFileResponse(path string) GeminiResponse {
	file, err := os.Open(path)
	if err != nil {
		return GeminiResponse{
			statusCode: 51,
			errorMsg:   "File not found",
		}
	}
	defer file.Close()

	sniffedContentType, err := sniffContentType(file)
	if err != nil {
		return GeminiResponse{
			statusCode: 40,
			errorMsg:   "Unexpected error",
		}
	}

	content := new(strings.Builder)
	buf := make([]byte, 1024)
	_, err = file.Seek(0, 0)
	if err != nil {
		return GeminiResponse{
			statusCode: 40,
			errorMsg:   "Unexpected error",
		}
	}

	for {
		n, err := file.Read(buf)
		if err != nil {
			break
		}
		content.Write(buf[:n])
	}

	return GeminiResponse{
		statusCode: 20,
		mimeType:   sniffedContentType,
		content:    content.String(),
	}
}

func sniffContentType(file *os.File) (string, error) {
	_, err := file.Seek(0, 0)
	if err != nil {
		return "", err
	}

	sniffData := make([]byte, 512)
	_, err = file.Read(sniffData)
	if err != nil {
		return "", err
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return "", err
	}

	detectedType := http.DetectContentType(sniffData)
	if detectedType == "application/octet-stream" && filepath.Ext(file.Name()) == ".gmi" {
		return "text/gemini", nil
	}

	return detectedType, nil
}
