package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/go-fitz"
	"github.com/nguyenthenguyen/docx"
)

func ParseFile(path string) (string, string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	title := strings.TrimSuffix(filepath.Base(path), ext)
	switch ext {
	case ".md", ".txt":
		content, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		text := string(content)
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimPrefix(line, "# ")
				break
			}
		}
		return title, text, nil
	case ".pdf":
		doc, err := fitz.New(path)
		if err != nil {
			return "", "", err
		}
		defer doc.Close()
		var b strings.Builder
		for i := 0; i < doc.NumPage(); i++ {
			text, _ := doc.Text(i)
			b.WriteString(text)
			b.WriteString("\n\n")
		}
		return title, strings.TrimSpace(b.String()), nil
	case ".docx":
		r, err := docx.ReadDocxFile(path)
		if err != nil {
			return "", "", err
		}
		defer r.Close()
		return title, strings.TrimSpace(r.Editable().GetContent()), nil
	default:
		return "", "", fmt.Errorf("unsupported: %s", ext)
	}
}

func IsSupported(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".md" || ext == ".txt" || ext == ".pdf" || ext == ".docx"
}
