package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 应用结构体
type App struct {
	ctx     context.Context
	handler *APIHandler
	search  *SearchService
}

// NewApp 创建应用实例
func NewApp(handler *APIHandler, search *SearchService) *App {
	return &App{
		handler: handler,
		search:  search,
	}
}

// OnStartup 应用启动时调用
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	runtime.LogInfo(ctx, "AI笔记助手 启动")

	// Initialize search engine (Qdrant + BM25 + embedding)
	go func() {
		if err := a.search.Init(); err != nil {
			fmt.Printf("[App] Search init warning: %v\n", err)
		}

		// Run incremental update on startup
		changed, err := a.search.IncrementalUpdate()
		if err != nil {
			fmt.Printf("[App] Incremental update error: %v\n", err)
		} else if changed > 0 {
			fmt.Printf("[App] Incremental update: %d notes updated\n", changed)
		}
	}()
}

// OnBeforeClose 应用关闭前调用
func (a *App) OnBeforeClose(ctx context.Context) bool {
	a.search.Shutdown()
	return false
}

// ============ 笔记相关方法 (暴露给前端) ============

func (a *App) GetAllNotes() ([]Note, error) {
	return a.handler.GetAllNotes()
}

func (a *App) CreateNote(title, content, category, tags string) (*Note, error) {
	return a.handler.CreateNote(CreateNoteReq{
		Title:    title,
		Content:  content,
		Category: category,
		Tags:     tags,
	})
}

func (a *App) UpdateNote(id int64, title, content, category, tags string) (*Note, error) {
	req := UpdateNoteReq{ID: id}
	req.Title = &title
	req.Content = &content
	req.Category = &category
	req.Tags = &tags
	return a.handler.UpdateNote(req)
}

func (a *App) DeleteNote(id int64) error {
	return a.handler.DeleteNote(id)
}

func (a *App) GetNoteCategories() ([]string, error) {
	return a.handler.GetNoteCategories()
}

// ============ AI 对话相关方法 (暴露给前端) ============

func (a *App) GetChatHistory() ([]ChatMessage, error) {
	return a.handler.GetChatHistory()
}

func (a *App) SendChatMessage(userMessage string) (*ChatResponse, error) {
	return a.handler.ChatWithAI(userMessage)
}

func (a *App) ClearChatHistory() error {
	return a.handler.ClearChatHistory()
}

func (a *App) DeleteChatMessage(id int64) error {
	return a.handler.DeleteChatMessage(id)
}

// ============ 设置相关方法 (暴露给前端) ============

func (a *App) GetSetting(key string) (string, error) {
	return a.handler.GetSetting(key)
}

func (a *App) SetSetting(key, value string) error {
	return a.handler.SetSetting(key, value)
}

func (a *App) GetAllSettings() (map[string]string, error) {
	return a.handler.GetAllSettings()
}

// ============ 搜索引擎相关方法 (暴露给前端) ============

func (a *App) CheckLeannStatus() *LeannStatus {
	return a.handler.CheckLeannStatus()
}

func (a *App) RebuildIndex() error {
	return a.handler.RebuildIndex()
}

func (a *App) RebuildIndexIfChanged() (bool, string, error) {
	return a.handler.RebuildIndexIfChanged()
}

func (a *App) TestLLMConnection() *TestConnectionResult {
	return a.handler.TestLLMConnection()
}

func (a *App) GetAppDir() string {
	return a.handler.GetAppDir()
}

func (a *App) ImportFolder(folderPath string) (*ImportResult, error) {
	return a.handler.ImportFolder(folderPath)
}

func (a *App) SelectFolder() (*ImportResult, error) {
	folderPath, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择文件夹"})
	if err != nil {
		return nil, err
	}
	if folderPath == "" {
		return nil, nil
	}
	return a.handler.ImportFolder(folderPath)
}
