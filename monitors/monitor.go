package monitors

import (
	"os"
	"time"
)

type FileMonitor struct {
	// tbh this should probably emit a single event with all the files that changed instead
	Changed   chan string
	fileStats map[string]os.FileInfo
}

func NewFileMonitor(filePaths []string) (*FileMonitor, error) {
	fileStats := make(map[string]os.FileInfo)
	m := &FileMonitor{
		fileStats: fileStats,
		Changed:   make(chan string),
	}

	for _, filePath := range filePaths {
		stat, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}

		m.fileStats[filePath] = stat
	}

	go m.watch()

	return m, nil
}

func (m *FileMonitor) AddFile(filePath string) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	m.fileStats[filePath] = stat
	m.Changed <- filePath
	return nil
}

func (m *FileMonitor) watch() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	m.Changed <- "init"

	for range ticker.C {
		m.checkFiles()
	}
}

func (m *FileMonitor) checkFiles() {
	fileChanged := false
	for filePath, stat := range m.fileStats {
		newStat, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		if newStat.Size() != stat.Size() || newStat.ModTime() != stat.ModTime() {
			fileChanged = true
			m.fileStats[filePath] = newStat
		}
	}

	if fileChanged {
		m.Changed <- "file"
	}
}
