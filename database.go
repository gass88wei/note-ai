package main

import (
	"database/sql"
	"os"
	"path/filepath"

	"note-ai/internal/indexer"

	_ "modernc.org/sqlite"
)

type Database struct {
	db     *sql.DB
	appDir string
}

// Note 表示一条笔记
type Note struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	Tags      string `json:"tags"` // 逗号分隔的标签
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ChatMessage 表示对话消息
type ChatMessage struct {
	ID         int64  `json:"id"`
	Role       string `json:"role"` // user, assistant, system
	Content    string `json:"content"`
	Timestamp  string `json:"timestamp"`
	NoteIds    string `json:"note_ids"`    // 搜索到的相关笔记ID, 逗号分隔
	QuestionId int64  `json:"question_id"` // 关联的问题ID(对assistant消息)
}

// Setting 表示应用设置
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func NewDatabase() (*Database, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	appDir := filepath.Join(homeDir, ".note-ai")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(appDir, "notes.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent performance
	db.Exec("PRAGMA journal_mode=WAL")

	database := &Database{
		db:     db,
		appDir: appDir,
	}

	if err := database.init(); err != nil {
		return nil, err
	}

	return database, nil
}

func (d *Database) GetAppDir() string {
	return d.appDir
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL DEFAULT '',
		category TEXT DEFAULT '未分类',
		tags TEXT DEFAULT '',
		content_hash TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_note_category ON notes(category);
	CREATE INDEX IF NOT EXISTS idx_note_created_at ON notes(created_at DESC);

	CREATE TABLE IF NOT EXISTS chat_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		note_ids TEXT DEFAULT '',
		question_id INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_msg_role ON chat_messages(role);
	CREATE INDEX IF NOT EXISTS idx_msg_timestamp ON chat_messages(timestamp DESC);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);

	INSERT OR IGNORE INTO settings (key, value) VALUES ('leann_index_path', '');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('llm_base_url', 'http://localhost:11434/v1');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('llm_model', 'qwen2.5');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('llm_api_key', '');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('ai_top_k', '5');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('ai_system_prompt', '你是一个个人笔记助手。基于用户提供的笔记内容回答问题。如果笔记中没有相关信息，请如实告知用户。');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('embed_base_url', 'http://127.0.0.1:1234/v1');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('embed_model', '');
	INSERT OR IGNORE INTO settings (key, value) VALUES ('embed_api_key', '');

	CREATE TABLE IF NOT EXISTS chunks (
		chunk_id INTEGER PRIMARY KEY,
		note_id INTEGER NOT NULL,
		text TEXT NOT NULL,
		section_id INTEGER DEFAULT 0,
		chunk_idx INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_chunks_note_id ON chunks(note_id);
	`

	_, err := d.db.Exec(schema)
	if err != nil {
		return err
	}

	// Migration: add content_hash column if missing
	d.db.Exec(`ALTER TABLE notes ADD COLUMN content_hash TEXT DEFAULT ''`)

	return nil
}

// ============ Note CRUD ============

func (d *Database) GetAllNotes() ([]Note, error) {
	rows, err := d.db.Query(`
		SELECT id, title, content, category, tags, created_at, updated_at
		FROM notes ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.Category, &n.Tags, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (d *Database) GetNoteByID(id int64) (*Note, error) {
	var n Note
	err := d.db.QueryRow(`
		SELECT id, title, content, category, tags, created_at, updated_at
		FROM notes WHERE id = ?
	`, id).Scan(&n.ID, &n.Title, &n.Content, &n.Category, &n.Tags, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (d *Database) CreateNote(note *Note) (*Note, error) {
	result, err := d.db.Exec(
		`INSERT INTO notes (title, content, category, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		note.Title, note.Content, note.Category, note.Tags, note.CreatedAt, note.UpdatedAt)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	note.ID = id
	return note, nil
}

func (d *Database) UpdateNote(note *Note) error {
	_, err := d.db.Exec(
		`UPDATE notes SET title = ?, content = ?, category = ?, tags = ?, updated_at = ? WHERE id = ?`,
		note.Title, note.Content, note.Category, note.Tags, note.UpdatedAt, note.ID)
	return err
}

func (d *Database) DeleteNote(id int64) error {
	_, err := d.db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}

func (d *Database) GetNoteCategories() ([]string, error) {
	rows, err := d.db.Query(`SELECT DISTINCT category FROM notes ORDER BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

// ============ Chat Messages ============

func (d *Database) GetAllChatMessages() ([]ChatMessage, error) {
	rows, err := d.db.Query(`
		SELECT id, role, content, timestamp, note_ids, question_id
		FROM chat_messages ORDER BY timestamp ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.Timestamp, &m.NoteIds, &m.QuestionId)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (d *Database) CreateChatMessage(msg *ChatMessage) (*ChatMessage, error) {
	result, err := d.db.Exec(
		`INSERT INTO chat_messages (role, content, note_ids, question_id) VALUES (?, ?, ?, ?)`,
		msg.Role, msg.Content, msg.NoteIds, msg.QuestionId)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	msg.ID = id
	return msg, nil
}

func (d *Database) ClearChatMessages() error {
	_, err := d.db.Exec(`DELETE FROM chat_messages`)
	return err
}

func (d *Database) DeleteChatMessage(id int64) error {
	_, err := d.db.Exec(`DELETE FROM chat_messages WHERE id = ?`, id)
	return err
}

// ============ Settings ============

func (d *Database) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (d *Database) SetSetting(key, value string) error {
	_, err := d.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	return err
}

func (d *Database) GetAllSettings() (map[string]string, error) {
	rows, err := d.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, rows.Err()
}

// ============ Content Hash (for incremental index) ============

func (d *Database) SetContentHash(noteID int64, hash string) {
	d.db.Exec(`UPDATE notes SET content_hash = ? WHERE id = ?`, hash, noteID)
}

func (d *Database) DeleteContentHash(noteID int64) {
	d.db.Exec(`UPDATE notes SET content_hash = '' WHERE id = ?`, noteID)
}

func (d *Database) GetAllContentHashes() (map[int64]string, error) {
	rows, err := d.db.Query(`SELECT id, content_hash FROM notes WHERE content_hash != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashes := make(map[int64]string)
	for rows.Next() {
		var id int64
		var hash string
		if err := rows.Scan(&id, &hash); err != nil {
			continue
		}
		hashes[id] = hash
	}
	return hashes, rows.Err()
}

// ============ Chunks (for vector index) ============

// ChunkRow represents a chunk stored in SQLite.
type ChunkRow struct {
	ChunkID   int64
	NoteID    int64
	Text      string
	SectionID int
	ChunkIdx  int
}

// SaveChunks inserts or replaces chunks for a note.
func (d *Database) SaveChunks(chunks []indexer.Chunk) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO chunks (chunk_id, note_id, text, section_id, chunk_idx) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, c := range chunks {
		_, err := stmt.Exec(c.ID, c.NoteID, c.Text, c.SectionID, c.Index)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// DeleteNoteChunks removes all chunks for a note.
func (d *Database) DeleteNoteChunks(noteID int64) error {
	_, err := d.db.Exec(`DELETE FROM chunks WHERE note_id = ?`, noteID)
	return err
}

// ClearChunks removes all chunks.
func (d *Database) ClearChunks() error {
	_, err := d.db.Exec(`DELETE FROM chunks`)
	return err
}

// GetChunksByIDs returns chunk texts for the given chunk IDs.
func (d *Database) GetChunksByIDs(ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Build placeholders
	placeholders := ""
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}

	rows, err := d.db.Query("SELECT chunk_id, text FROM chunks WHERE chunk_id IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			continue
		}
		result[id] = text
	}
	return result, rows.Err()
}
