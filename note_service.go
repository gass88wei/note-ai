package main

import (
	"fmt"
	"strings"
	"time"

	"note-ai/internal/parser"
	"note-ai/internal/scanner"
)

// NoteService 笔记业务逻辑
type NoteService struct {
	db     *Database
	search *SearchService
	llm    *LLMClient
}

func NewNoteService(db *Database, search *SearchService, llm *LLMClient) *NoteService {
	return &NoteService{
		db:     db,
		search: search,
		llm:    llm,
	}
}

// GetAllNotes 获取所有笔�?
func (s *NoteService) GetAllNotes() ([]Note, error) {
	return s.db.GetAllNotes()
}

// CreateNote 创建笔记并增量更新索�?
func (s *NoteService) CreateNote(title, content, category, tags string) (*Note, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	note := &Note{
		Title:     title,
		Content:   content,
		Category:  category,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	note, err := s.db.CreateNote(note)
	if err != nil {
		return nil, err
	}

	// 增量索引：直接追�?
	go s.search.IndexNoteAdded(note)

	return note, nil
}

// UpdateNote 更新笔记并增量更新索�?
func (s *NoteService) UpdateNote(id int64, title, content, category, tags string) (*Note, error) {
	note, err := s.db.GetNoteByID(id)
	if err != nil {
		return nil, err
	}

	note.Title = title
	note.Content = content
	note.Category = category
	note.Tags = tags
	note.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	if err := s.db.UpdateNote(note); err != nil {
		return nil, err
	}

	// 增量索引：替换索引条�?
	go s.search.IndexNoteUpdated(note)

	return note, nil
}

// DeleteNote 删除笔记并从索引移除
func (s *NoteService) DeleteNote(id int64) error {
	// 增量索引：从索引移除
	go s.search.IndexNoteDeleted(id)

	return s.db.DeleteNote(id)
}

// GetNoteCategories 获取所有分�?
func (s *NoteService) GetNoteCategories() ([]string, error) {
	return s.db.GetNoteCategories()
}

// AskQuestion 基于笔记搜索 + LLM 回答问题
type QuestionResult struct {
	Answer        string         `json:"answer"`
	NoteIds       []int64        `json:"note_ids"`
	SearchResults []SearchResult `json:"search_results"`
}

func (s *NoteService) AskQuestion(question string) (*QuestionResult, error) {
	topKStr, _ := s.db.GetSetting("ai_top_k")
	topK := 5
	fmt.Sscanf(topKStr, "%d", &topK)

	searchResults, err := s.search.Search(question, topK)
	if err != nil {
		return nil, fmt.Errorf("搜索笔记失败: %v", err)
	}

	if len(searchResults) == 0 {
		return &QuestionResult{
			Answer: "未在笔记中找到相关信息。",
		}, nil
	}

	answer, err := s.llm.GetAnswer(searchResults, question)
	if err != nil {
		return nil, fmt.Errorf("获取 AI 回答失败: %v", err)
	}

	var noteIds []int64
	for _, r := range searchResults {
		if r.NoteID > 0 {
			noteIds = append(noteIds, r.NoteID)
		}
	}

	return &QuestionResult{
		Answer:        answer,
		NoteIds:       noteIds,
		SearchResults: searchResults,
	}, nil
}

// ChatWithAI 完整�?AI 对话流程
func (s *NoteService) ChatWithAI(userMessage string) ([]ChatMessage, error) {
	userMsg := &ChatMessage{
		Role:      "user",
		Content:   userMessage,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	userMsg, err := s.db.CreateChatMessage(userMsg)
	if err != nil {
		return nil, err
	}

	result, err := s.AskQuestion(userMessage)
	if err != nil {
		errorMsg := &ChatMessage{
			Role:      "assistant",
			Content:   "回答失败: " + err.Error(),
			Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		}
		s.db.CreateChatMessage(errorMsg)
		return nil, err
	}

	noteIdStrs := make([]string, len(result.NoteIds))
	for i, id := range result.NoteIds {
		noteIdStrs[i] = fmt.Sprintf("%d", id)
	}

	assistantMsg := &ChatMessage{
		Role:       "assistant",
		Content:    result.Answer,
		Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
		NoteIds:    strings.Join(noteIdStrs, ","),
		QuestionId: userMsg.ID,
	}
	assistantMsg, err = s.db.CreateChatMessage(assistantMsg)
	if err != nil {
		return nil, err
	}

	return []ChatMessage{*userMsg, *assistantMsg}, nil
}

// GetChatHistory 获取聊天历史
func (s *NoteService) GetChatHistory() ([]ChatMessage, error) {
	return s.db.GetAllChatMessages()
}

// ClearChatHistory 清空聊天历史
func (s *NoteService) ClearChatHistory() error {
	return s.db.ClearChatMessages()
}

// DeleteChatMessage 删除单条消息
func (s *NoteService) DeleteChatMessage(id int64) error {
	return s.db.DeleteChatMessage(id)
}

func (s *NoteService) ImportFolder(folderPath string) (int, error) {
	files, err := scanner.ScanFolder(folderPath)
	if err != nil {
		return 0, err
	}
	imported := 0
	for _, f := range files {
		title, content, err := parser.ParseFile(f.Path)
		if err != nil {
			continue
		}
		s.CreateNote(title, content, "导入", "")
		imported++
	}
	return imported, nil
}
