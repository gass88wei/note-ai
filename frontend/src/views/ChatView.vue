<script setup lang="ts">
import { ref, nextTick, onMounted } from 'vue'
import { marked } from 'marked'
import type { ChatMessage } from '@/types'
import { GetChatHistory, SendChatMessage, ClearChatHistory, DeleteChatMessage } from '../../wailsjs/go/main/App'

const messages = ref<ChatMessage[]>([])
const input = ref('')
const messageList = ref<HTMLElement | null>(null)
const isSending = ref(false)

marked.setOptions({
  breaks: true,
  gfm: true
})

onMounted(async () => {
  await loadHistory()
})

async function loadHistory() {
  try {
    const history = await GetChatHistory() || []
    messages.value = history
  } catch (err) {
    console.error('加载失败:', err)
    messages.value = []
  }
}

async function sendMessage() {
  if (!input.value.trim() || isSending.value) return

  const userMessage = input.value.trim()
  input.value = ''
  isSending.value = true

  // 立即显示用户消息
  const tempId = Date.now()
  messages.value.push({
    id: tempId,
    role: 'user',
    content: userMessage,
    timestamp: new Date().toLocaleString('zh-CN'),
    note_ids: ''
  })
  scrollToBottom()

  try {
    await SendChatMessage(userMessage)
    await loadHistory()
    scrollToBottom()
  } catch (err) {
    // 出错时也刷新历史，因为后端可能已保存了错误消息到数据库
    await loadHistory()
    scrollToBottom()
  } finally {
    isSending.value = false
  }
}

async function clearChat() {
  if (!confirm('确定清空聊天历史吗？此操作不可恢复。')) return
  
  try {
    await ClearChatHistory()
    messages.value = []
  } catch (err) {
    alert('清空失败:' + err)
  }
}

const copyToClipboard = async (content: string) => {
  try {
    await navigator.clipboard.writeText(content)
    alert('已复制到剪贴板')
  } catch (err) {
    alert('复制失败')
  }
}

const deleteMessage = async (id: number) => {
  if (!confirm('确定删除这条消息吗？')) return
  try {
    await DeleteChatMessage(id)
    await loadHistory()
  } catch (err) {
    alert('删除失败:' + err)
  }
}

const scrollToBottom = () => {
  setTimeout(() => {
    if (messageList.value) {
      messageList.value.scrollTop = messageList.value.scrollHeight
    }
  }, 100)
}
</script>

<template>
  <div class="chat-container">
    <!-- Header -->
    <div class="chat-header">
      <div>
        <h2 class="title">AI 对话</h2>
        <p class="subtitle">基于笔记内容智能问答</p>
      </div>
      <button @click="clearChat" class="btn-danger">🗑️ 清空聊天</button>
    </div>

    <!-- Messages -->
    <div ref="messageList" class="messages-list">
      <div v-for="msg in messages" :key="msg.id" class="message-wrapper" :class="msg.role">
        <div class="message-bubble">
          <div class="message-content" :class="{ 'user-msg': msg.role === 'user' }" v-html="msg.role === 'assistant' ? marked(msg.content || '') : msg.content"></div>
          <div class="message-actions">
            <button @click="copyToClipboard(msg.content)" class="action-btn" title="复制">
              📋 复制
            </button>
            <button @click="deleteMessage(msg.id)" class="action-btn" title="删除">
              🗑️ 删除
            </button>
            <span class="message-time">{{ msg.timestamp?.substring(11, 19) }}</span>
          </div>
        </div>
      </div>

      <div v-if="isSending" class="message-wrapper assistant">
        <div class="message-bubble">
          <div class="message-content">思考中...</div>
        </div>
      </div>

      <div v-if="messages.length === 0 && !isSending" class="empty-chat">
        <div class="empty-icon">💬</div>
        <p class="empty-text">开始与 AI 对话</p>
        <p class="empty-subtext">AI 会基于你的笔记内容回答问题</p>
      </div>
    </div>

    <!-- Input -->
    <div class="input-area">
      <div class="input-wrapper">
        <input
          v-model="input"
          @keyup.enter="sendMessage"
          placeholder="输入问题..."
          :disabled="isSending"
          class="input-field"
        />
        <button
          @click="sendMessage"
          :disabled="isSending || !input.trim()"
          class="send-button"
        >
          发送
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background-color: #111827;
}

/* Header */
.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #374151;
  background-color: #1f2937;
}

.title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0 0 0.25rem 0;
}

.subtitle {
  font-size: 0.875rem;
  color: #9ca3af;
  margin: 0;
}

/* Messages */
.messages-list {
  flex: 1;
  overflow-y: auto;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.message-wrapper {
  display: flex;
  clear: both;
}

.message-wrapper.user {
  justify-content: flex-end;
}

.message-wrapper.assistant {
  justify-content: flex-start;
}

.message-bubble {
  max-width: 75%;
  padding: 0.75rem 1rem;
  border-radius: 0.75rem;
  word-wrap: break-word;
}

.message-wrapper.user .message-bubble {
  background-color: #0d9488;
  color: white;
  border-bottom-right-radius: 0.25rem;
}

.message-wrapper.assistant .message-bubble {
  background-color: #1f2937;
  color: #f9fafb;
  border: 1px solid #374151;
  border-bottom-left-radius: 0.25rem;
}

.message-content {
  font-size: 0.875rem;
  line-height: 1.5;
  margin-bottom: 0.5rem;
  white-space: pre-wrap;
  word-break: break-word;
}

.message-content :deep(h1),
.message-content :deep(h2),
.message-content :deep(h3) {
  margin: 0.5rem 0;
  font-weight: 600;
}

.message-content :deep(p) {
  margin: 0.5rem 0;
}

.message-content :deep(ul),
.message-content :deep(ol) {
  margin: 0.5rem 0;
  padding-left: 1.5rem;
}

.message-content :deep(li) {
  margin: 0.25rem 0;
}

.message-content :deep(code) {
  background-color: rgba(0,0,0,0.3);
  padding: 0.125rem 0.25rem;
  border-radius: 0.25rem;
  font-size: 0.8em;
}

.message-content :deep(pre) {
  background-color: rgba(0,0,0,0.3);
  padding: 0.5rem;
  border-radius: 0.375rem;
  overflow-x: auto;
}

.message-content :deep(pre code) {
  background: none;
  padding: 0;
}

.message-content :deep(blockquote) {
  border-left: 3px solid #0d9488;
  margin: 0.5rem 0;
  padding-left: 0.75rem;
  color: #9ca3af;
}

.message-content :deep(strong) {
  font-weight: 600;
  color: #fbbf24;
}

.message-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border-top: 1px solid rgba(255,255,255,0.1);
  padding-top: 0.5rem;
  margin-top: 0.5rem;
}

.message-wrapper.assistant .message-actions {
  border-top-color: rgba(255,255,255,0.1);
}

.action-btn {
  background: none;
  border: none;
  color: inherit;
  font-size: 0.75rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 0.25rem;
  opacity: 0.7;
  transition: all 0.2s;
}

.action-btn:hover {
  opacity: 1;
  background-color: rgba(255,255,255,0.1);
}

.message-time {
  font-size: 0.625rem;
  opacity: 0.6;
  margin-left: auto;
}

.empty-chat {
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

.empty-text {
  font-size: 1.125rem;
  margin: 0 0 0.5rem 0;
}

.empty-subtext {
  font-size: 0.875rem;
  margin: 0;
}

/* Input */
.input-area {
  padding: 1.5rem;
  border-top: 1px solid #374151;
  background-color: #1f2937;
}

.input-wrapper {
  display: flex;
  gap: 0.5rem;
}

.input-field {
  flex: 1;
  padding: 0.75rem 1rem;
  background-color: #111827;
  border: 1px solid #374151;
  border-radius: 0.5rem;
  color: #f9fafb;
  font-size: 0.875rem;
}

.input-field:focus {
  outline: none;
  border-color: #0d9488;
}

.input-field:disabled {
  opacity: 0.5;
}

.send-button {
  padding: 0.75rem 1.5rem;
  background-color: #0d9488;
  color: white;
  border: none;
  border-radius: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.send-button:hover:not(:disabled) {
  background-color: #0f766e;
}

.send-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-danger {
  padding: 0.5rem 1rem;
  background-color: #dc2626;
  color: white;
  border: none;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-danger:hover {
  background-color: #b91c1c;
}
</style>