package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Watcher struct {
	service *NoteService
	search  *SearchService
}

func NewWatcher(s *NoteService, se *SearchService) *Watcher { return &Watcher{s, se} }
func (w *Watcher) Run(folder string) error {
	folder, _ = filepath.Abs(folder)
	watcher, _ := fsnotify.NewWatcher()
	defer watcher.Close()
	filepath.Walk(folder, func(p string, i os.FileInfo, e error) error {
		if e == nil && i.IsDir() && !strings.HasPrefix(i.Name(), ".") {
			watcher.Add(p)
		}
		return nil
	})
	fmt.Printf("📁 监听中: %s\n", folder)
	d := time.NewTimer(0)
	<-d.C
	pending := false
	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(ev.Name))
			if ext == ".md" || ext == ".txt" || ext == ".pdf" || ext == ".docx" {
				pending = true
				d.Reset(2 * time.Second)
			}
		case <-d.C:
			if pending {
				c, _ := w.search.IncrementalUpdate()
				fmt.Printf("更新 %d 条\n", c)
				pending = false
			}
		}
	}
}
