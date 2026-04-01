package main

import (
	"fmt"
	"strconv"
	"strings"
)

// APIHandler 处理所有 Wails 前端调用
type APIHandler struct {
	service *NoteService
	search  *SearchService
	llm     *LLMClient
	db      *Database
}

func NewAPIHandler(service *NoteService, search *SearchService, llm *LLMClient, db *Database) *APIHandler {
	return &APIHandler{
		service: service,
		search:  search,
		llm:     llm,
		db:      db,
	}
}

// ============ 笔记 API ============

func (h *APIHandler) GetAllNotes() ([]Note, error) {
	return h.service.GetAllNotes()
}

type CreateNoteReq struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Tags     string `json:"tags"`
}

func (h *APIHandler) CreateNote(req CreateNoteReq) (*Note, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("标题不能为空")
	}
	return h.service.CreateNote(req.Title, req.Content, req.Category, req.Tags)
}

type UpdateNoteReq struct {
	ID       int64   `json:"id"`
	Title    *string `json:"title,omitempty"`
	Content  *string `json:"content,omitempty"`
	Category *string `json:"category,omitempty"`
	Tags     *string `json:"tags,omitempty"`
}

func (h *APIHandler) UpdateNote(req UpdateNoteReq) (*Note, error) {
	title, content, category, tags := "", "", "", ""
	if req.Title != nil {
		title = *req.Title
	}
	if req.Content != nil {
		content = *req.Content
	}
	if req.Category != nil {
		category = *req.Category
	}
	if req.Tags != nil {
		tags = *req.Tags
	}
	return h.service.UpdateNote(req.ID, title, content, category, tags)
}

func (h *APIHandler) DeleteNote(id int64) error {
	return h.service.DeleteNote(id)
}

func (h *APIHandler) GetNoteCategories() ([]string, error) {
	return h.service.GetNoteCategories()
}

// ============ AI 对话 API ============

type QuestionResponse struct {
	Answer  string  `json:"answer"`
	NoteIds []int64 `json:"note_ids"`
}

func (h *APIHandler) AskQuestion(question string) (*QuestionResponse, error) {
	result, err := h.service.AskQuestion(question)
	if err != nil {
		return nil, err
	}
	return &QuestionResponse{
		Answer:  result.Answer,
		NoteIds: result.NoteIds,
	}, nil
}

type ChatResponse struct {
	UserInput string  `json:"user_input"`
	Answer    string  `json:"answer"`
	NoteIds   []int64 `json:"note_ids"`
}

func (h *APIHandler) ChatWithAI(userMessage string) (*ChatResponse, error) {
	messages, err := h.service.ChatWithAI(userMessage)
	if err != nil {
		return nil, err
	}

	var userInput, answer string
	var noteIds []int64

	for _, m := range messages {
		if m.Role == "user" {
			userInput = m.Content
		} else if m.Role == "assistant" {
			answer = m.Content
			if m.NoteIds != "" {
				for _, idStr := range strings.Split(m.NoteIds, ",") {
					idStr = strings.TrimSpace(idStr)
					if idStr == "" {
						continue
					}
					id, err := strconv.ParseInt(idStr, 10, 64)
					if err == nil && id > 0 {
						noteIds = append(noteIds, id)
					}
				}
			}
		}
	}

	return &ChatResponse{
		UserInput: userInput,
		Answer:    answer,
		NoteIds:   noteIds,
	}, nil
}

func (h *APIHandler) GetChatHistory() ([]ChatMessage, error) {
	return h.service.GetChatHistory()
}

func (h *APIHandler) ClearChatHistory() error {
	return h.service.ClearChatHistory()
}

func (h *APIHandler) DeleteChatMessage(id int64) error {
	return h.service.DeleteChatMessage(id)
}

// ============ 搜索引擎管理 API ============

type LeannStatus struct {
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

func (h *APIHandler) CheckLeannStatus() *LeannStatus {
	available, msg := h.search.CheckStatus()
	return &LeannStatus{
		Available: available,
		Message:   msg,
	}
}

func (h *APIHandler) RebuildIndex() error {
	return h.search.IndexAllNotes()
}

func (h *APIHandler) RebuildIndexIfChanged() (bool, string, error) {
	changed, err := h.search.IncrementalUpdate()
	if err != nil {
		return false, "", err
	}
	if changed == 0 {
		return false, "No changes detected", nil
	}
	return true, fmt.Sprintf("%d notes updated", changed), nil
}

// ============ 设置 API ============

func (h *APIHandler) GetSetting(key string) (string, error) {
	return h.db.GetSetting(key)
}

func (h *APIHandler) SetSetting(key, value string) error {
	return h.db.SetSetting(key, value)
}

func (h *APIHandler) GetAllSettings() (map[string]string, error) {
	return h.db.GetAllSettings()
}

// TestLLMConnection 测试 LLM API 连接
type TestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (h *APIHandler) TestLLMConnection() *TestConnectionResult {
	success, msg := h.llm.TestConnection()
	return &TestConnectionResult{
		Success: success,
		Message: msg,
	}
}

func (h *APIHandler) GetAppDir() string {
	return h.db.GetAppDir()
}
