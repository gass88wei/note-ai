<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Note } from '@/types'
import { GetAllNotes, GetNoteCategories, CreateNote, UpdateNote, DeleteNote } from '../../wailsjs/go/main/App'

const searchQuery = ref('')
const selectedCategory = ref('')
const showEditor = ref(false)
const editingNote = ref<Note | null>(null)
const newNoteTitle = ref('')
const newNoteCategory = ref('未分类')
const showNewNoteModal = ref(false)
const notes = ref<Note[]>([])
const categories = ref<string[]>(['未分类'])

onMounted(async () => {
  await loadNotes()
  await loadCategories()
})

async function loadNotes() {
  try {
    notes.value = await GetAllNotes() || []
  } catch (err) {
    console.error('加载失败:', err)
    notes.value = []
  }
}

async function loadCategories() {
  try {
    const cats = await GetNoteCategories() || []
    categories.value = ['未分类', ...cats.filter(c => c !== '未分类')]
  } catch (err) {
    console.error('加载分类失败:', err)
  }
}

const filteredNotes = computed(() => {
  let list = notes.value
  
  if (selectedCategory.value) {
    list = list.filter(n => n.category === selectedCategory.value)
  }
  
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    list = list.filter(n => 
      n.title.toLowerCase().includes(q) || 
      n.content?.toLowerCase().includes(q) ||
      n.tags?.toLowerCase().includes(q)
    )
  }
  
  return list
})

const openNewNote = () => {
  showNewNoteModal.value = true
  newNoteTitle.value = ''
  newNoteCategory.value = categories.value[0] || '未分类'
}

const createNote = async () => {
  if (!newNoteTitle.value.trim()) return
  
  try {
    const note = await CreateNote(newNoteTitle.value, '', newNoteCategory.value, '')
    if (note) {
      showNewNoteModal.value = false
      editNote(note)
      await loadNotes()
    }
  } catch (err) {
    alert('创建失败:' + err)
  }
}

const editNote = (note: Note) => {
  editingNote.value = note
  showEditor.value = true
}

const saveNote = async (data: any) => {
  try {
    await UpdateNote(data.id, data.title, data.content, data.category, data.tags)
    showEditor.value = false
    editingNote.value = null
    await loadNotes()
  } catch (err) {
    alert('保存失败:' + err)
  }
}

const deleteNoteFromList = async (id: number, event: Event) => {
  event.stopPropagation()
  if (!confirm('确定删除这篇笔记吗？')) return
  
  try {
    await DeleteNote(id)
    await loadNotes()
    if (editingNote.value?.id === id) {
      showEditor.value = false
      editingNote.value = null
    }
  } catch (err) {
    alert('删除失败:' + err)
  }
}

const deleteNoteInEditor = async (id: number) => {
  if (!confirm('确定删除这篇笔记吗？')) return
  
  try {
    await DeleteNote(id)
    showEditor.value = false
    editingNote.value = null
    await loadNotes()
  } catch (err) {
    alert('删除失败:' + err)
  }
}

const cancelEdit = () => {
  showEditor.value = false
  editingNote.value = null
}
</script>

<template>
  <div class="notes-container">
    <!-- 左侧列表 -->
    <div class="notes-sidebar">
      <div class="sidebar-header">
        <h2 class="title">笔记</h2>
        <button @click="openNewNote" class="btn-primary">+ 新建</button>
      </div>

      <input
        v-model="searchQuery"
        placeholder="搜索笔记..."
        class="search-input"
      />

      <select v-model="selectedCategory" class="category-select">
        <option value="">全部分类</option>
        <option v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</option>
      </select>

      <div class="notes-list">
        <div
          v-for="note in filteredNotes"
          :key="note.id"
          @click="editNote(note)"
          class="note-item"
          :class="{ active: editingNote?.id === note.id }"
        >
          <div class="note-content">
            <h3 class="note-title">{{ note.title }}</h3>
            <p class="note-preview">{{ note.content?.substring(0, 50) }}</p>
            <div class="note-meta">
              <span class="note-category">{{ note.category }}</span>
              <span class="note-date">{{ note.updated_at?.substring(0, 10) }}</span>
            </div>
          </div>
          <button 
            @click.stop="deleteNoteFromList(note.id, $event)"
            class="btn-delete"
            title="删除笔记"
          >
            🗑️
          </button>
        </div>

        <div v-if="filteredNotes.length === 0" class="empty-state">
          <p>暂无笔记</p>
        </div>
      </div>
    </div>

    <!-- 右侧编辑器 -->
    <div class="notes-editor">
      <div v-if="showEditor && editingNote" class="editor-content">
        <input
          v-model="editingNote.title"
          placeholder="笔记标题"
          class="editor-title"
        />

        <div class="editor-meta">
          <div class="meta-field">
            <label>分类 (可直接输入新分类)</label>
            <input 
              v-model="editingNote.category" 
              list="categories-list" 
              class="meta-input" 
              placeholder="输入或选择分类"
            />
            <datalist id="categories-list">
              <option v-for="cat in categories" :key="cat" :value="cat" />
            </datalist>
          </div>
          <div class="meta-field">
            <label>标签</label>
            <input v-model="editingNote.tags" placeholder="tag1, tag2, tag3" class="meta-input" />
          </div>
        </div>

        <textarea
          v-model="editingNote.content"
          rows="30"
          placeholder="在这里写笔记..."
          class="editor-textarea"
        ></textarea>

        <div class="editor-actions">
          <button @click="deleteNoteInEditor(editingNote.id)" class="btn-danger">🗑️ 删除</button>
          <button @click="cancelEdit" class="btn-secondary">取消</button>
          <button @click="saveNote(editingNote)" class="btn-primary">💾 保存</button>
        </div>
      </div>

      <div v-else class="empty-editor">
        <div class="empty-icon">📝</div>
        <p>选择或创建笔记</p>
        <button @click="openNewNote" class="btn-link">新建笔记 →</button>
      </div>
    </div>

    <!-- 新建模态框 -->
    <div v-if="showNewNoteModal" class="modal-overlay" @click.self="showNewNoteModal = false">
      <div class="modal">
        <h3 class="modal-title">新建笔记</h3>
        
        <div class="form-group">
          <label>标题</label>
          <input
            v-model="newNoteTitle"
            @keyup.enter="createNote"
            placeholder="输入笔记标题..."
            class="input-field"
            autofocus
          />
        </div>
        
        <div class="form-group">
          <label>分类</label>
          <select v-model="newNoteCategory" class="input-field">
            <option v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</option>
          </select>
        </div>

        <div class="modal-actions">
          <button @click="showNewNoteModal = false" class="btn-secondary">取消</button>
          <button @click="createNote" :disabled="!newNoteTitle.trim()" class="btn-primary">创建</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.notes-container {
  display: flex;
  height: 100%;
  width: 100%;
}

/* Sidebar */
.notes-sidebar {
  width: 320px;
  background-color: #1f2937;
  border-right: 1px solid #374151;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid #374151;
}

.title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.search-input,
.category-select {
  width: calc(100% - 2rem);
  margin: 0.75rem 1rem;
  padding: 0.5rem 0.75rem;
  background-color: #111827;
  border: 1px solid #374151;
  border-radius: 0.375rem;
  color: #f9fafb;
  font-size: 0.875rem;
}

.notes-list {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem;
}

.note-item {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.75rem;
  margin-bottom: 0.5rem;
  background-color: #111827;
  border-radius: 0.375rem;
  cursor: pointer;
  transition: all 0.2s;
}

.note-item:hover {
  background-color: #374151;
}

.note-item.active {
  background-color: #0d9488;
}

.note-content {
  flex: 1;
  min-width: 0;
}

.note-title {
  font-weight: 600;
  margin: 0 0 0.25rem 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 0.875rem;
}

.note-preview {
  font-size: 0.75rem;
  color: #9ca3af;
  margin: 0 0 0.5rem 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.note-meta {
  display: flex;
  justify-content: space-between;
  font-size: 0.625rem;
  color: #6b7280;
}

.btn-delete {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1rem;
  padding: 0.25rem;
  opacity: 0.6;
  transition: all 0.2s;
  flex-shrink: 0;
}

.btn-delete:hover {
  opacity: 1;
  transform: scale(1.1);
}

.empty-state {
  text-align: center;
  padding: 2rem;
  color: #6b7280;
}

/* Editor */
.notes-editor {
  flex: 1;
  background-color: #111827;
  padding: 1.5rem;
  overflow-y: auto;
}

.editor-content {
  max-width: 800px;
  margin: 0 auto;
}

.editor-title {
  width: 100%;
  padding: 0.75rem;
  background-color: #1f2937;
  border: 1px solid #374151;
  border-radius: 0.375rem;
  color: #f9fafb;
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 1rem;
}

.editor-meta {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  margin-bottom: 1rem;
}

.meta-field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.meta-field label {
  font-size: 0.75rem;
  color: #9ca3af;
}

.meta-input {
  padding: 0.5rem 0.75rem;
  background-color: #1f2937;
  border: 1px solid #374151;
  border-radius: 0.375rem;
  color: #f9fafb;
}

.editor-textarea {
  width: 100%;
  min-height: 500px;
  padding: 0.75rem;
  background-color: #1f2937;
  border: 1px solid #374151;
  border-radius: 0.375rem;
  color: #f9fafb;
  font-family: monospace;
  font-size: 0.875rem;
  resize: vertical;
  line-height: 1.6;
}

.editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1rem;
}

.empty-editor {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #6b7280;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 1rem;
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background-color: #1f2937;
  padding: 1.5rem;
  border-radius: 0.5rem;
  width: 360px;
}

.modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0 0 1rem 0;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  font-size: 0.875rem;
  color: #9ca3af;
  margin-bottom: 0.25rem;
}

.input-field {
  width: 100%;
  padding: 0.5rem 0.75rem;
  background-color: #111827;
  border: 1px solid #374151;
  border-radius: 0.375rem;
  color: #f9fafb;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1.5rem;
}

/* Buttons */
.btn-primary,
.btn-secondary,
.btn-danger,
.btn-link {
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-primary {
  background-color: #0d9488;
  color: white;
}

.btn-primary:hover {
  background-color: #0f766e;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background-color: #374151;
  color: white;
}

.btn-secondary:hover {
  background-color: #4b5563;
}

.btn-danger {
  background-color: #dc2626;
  color: white;
}

.btn-danger:hover {
  background-color: #b91c1c;
}

.btn-link {
  background: none;
  color: #14b8a6;
  text-decoration: underline;
}
</style>
