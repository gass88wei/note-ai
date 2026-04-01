package scanner

import (
	"note-ai/internal/parser"
	"os"
	"path/filepath"
	"strings"
)

type ScanResult struct {
	Path, Title, Ext string
}

func ScanFolder(root string) ([]ScanResult, error) {
	var results []ScanResult
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if parser.IsSupported(ext) {
			results = append(results, ScanResult{path, strings.TrimSuffix(info.Name(), ext), ext})
		}
		return nil
	})
	return results, err
}
