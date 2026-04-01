// TypeScript 类型定义

export interface Note {
  id: number
  title: string
  content: string
  category: string
  tags: string
  created_at: string
  updated_at: string
}

export interface ChatMessage {
  id: number
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: string
  note_ids: string
  question_id: number
}

export interface Setting {
  key: string
  value: string
}

export interface ChatResponse {
  user_input: string
  answer: string
  note_ids: number[]
}

export interface LeannStatus {
  available: boolean
  message: string
}

export interface TestConnectionResult {
  success: boolean
  message: string
}
