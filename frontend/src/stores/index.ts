import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Note, ChatMessage, LeannStatus } from '@/types'
import * as api from '@/api'

export const useNoteStore = defineStore('note', () => {
  const notes = ref<Note[]>([])
  const selectedNote = ref<Note | null>(null)
  const loading = ref(false)
  const categories = ref<string[]>(['未分类'])

  async function loadNotes() {
    loading.value = true
    try {
      notes.value = await api.GetAllNotes()
    } catch (err) {
      console.error('加载笔记失败:', err)
    } finally {
      loading.value = false
    }
  }

  async function loadCategories() {
    try {
      categories.value = await api.GetNoteCategories()
    } catch (err) {
      console.error('加载分类失败:', err)
    }
  }

  async function createNote(title: string, content: string, category: string, tags: string) {
    const note = await api.CreateNote(title, content, category, tags)
    notes.value.unshift(note)
    return note
  }

  async function updateNote(id: number, title: string, content: string, category: string, tags: string) {
    const note = await api.UpdateNote(id, title, content, category, tags)
    const idx = notes.value.findIndex(n => n.id === id)
    if (idx !== -1) {
      notes.value[idx] = note
    }
    if (selectedNote.value?.id === id) {
      selectedNote.value = note
    }
    return note
  }

  async function deleteNote(id: number) {
    await api.DeleteNote(id)
    notes.value = notes.value.filter(n => n.id !== id)
    if (selectedNote.value?.id === id) {
      selectedNote.value = null
    }
  }

  function selectNote(note: Note | null) {
    selectedNote.value = note
  }

  return {
    notes,
    selectedNote,
    loading,
    categories,
    loadNotes,
    loadCategories,
    createNote,
    updateNote,
    deleteNote,
    selectNote,
  }
})

export const useChatStore = defineStore('chat', () => {
  const messages = ref<ChatMessage[]>([])
  const loading = ref(false)

  async function loadHistory() {
    try {
      messages.value = await api.GetChatHistory()
    } catch (err) {
      console.error('加载聊天历史失败:', err)
    }
  }

  async function sendMessage(userMessage: string) {
    loading.value = true
    try {
      const resp = await api.SendChatMessage(userMessage)
      // Refresh messages from server
      await loadHistory()
      return resp
    } catch (err) {
      console.error('发送消息失败:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function clearHistory() {
    await api.ClearChatHistory()
    messages.value = []
  }

  return {
    messages,
    loading,
    loadHistory,
    sendMessage,
    clearHistory,
  }
})

export const useSettingStore = defineStore('setting', () => {
  const settings = ref<Record<string, string>>({})
  const loading = ref(false)

  async function loadSettings() {
    loading.value = true
    try {
      settings.value = await api.GetAllSettings()
    } catch (err) {
      console.error('加载设置失败:', err)
    } finally {
      loading.value = false
    }
  }

  async function setSetting(key: string, value: string) {
    await api.SetSetting(key, value)
    settings.value[key] = value
  }

  async function checkLeann(): Promise<LeannStatus> {
    return await api.CheckLeannStatus()
  }

  async function rebuildIndex() {
    await api.RebuildIndex()
  }

  async function testLLM() {
    return await api.TestLLMConnection()
  }

  return {
    settings,
    loading,
    loadSettings,
    setSetting,
    checkLeann,
    rebuildIndex,
    testLLM,
  }
})
