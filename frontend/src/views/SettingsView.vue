<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { GetAllSettings, SetSetting, CheckLeannStatus, RebuildIndex, TestLLMConnection } from '../../wailsjs/go/main/App'

const form = ref({
  llm_base_url: 'http://localhost:11434/v1',
  llm_model: 'qwen2.5',
  llm_api_key: '',
  embed_base_url: 'http://127.0.0.1:1234/v1',
  embed_model: '',
  embed_api_key: '',
  ai_top_k: '5',
  ai_system_prompt: '你是一个个人笔记助手。基于用户提供的笔记内容回答问题。如果笔记中没有相关信息，请如实告知用户。',
})

const searchStatus = ref<string>('')
const searchStatusType = ref<'success' | 'error' | ''>('')
const llmTestResult = ref<string>('')
const llmTestStatusType = ref<'success' | 'error' | ''>('')
const isRebuilding = ref(false)

onMounted(async () => {
  await loadSettings()
})

async function loadSettings() {
  try {
    const settings = await GetAllSettings() || {}
    for (const key of Object.keys(form.value)) {
      if (settings[key]) {
        form.value[key as keyof typeof form.value] = settings[key]
      }
    }
  } catch (err) {
    console.error('加载设置失败:', err)
  }
}

async function saveSettings() {
  try {
    for (const key of Object.keys(form.value)) {
      await SetSetting(key, form.value[key as keyof typeof form.value])
    }
    alert('设置已保存!')
  } catch (err) {
    alert('保存失败:' + err)
  }
}

async function checkSearch() {
  searchStatusType.value = ''
  searchStatus.value = '正在检查...'
  try {
    const status = await CheckLeannStatus()
    if (status.available) {
      searchStatusType.value = 'success'
      searchStatus.value = '✅ ' + (status.message || '搜索引擎正常')
    } else {
      searchStatusType.value = 'error'
      searchStatus.value = '⚠️ ' + (status.message || '部分组件未就绪')
    }
  } catch (err) {
    searchStatusType.value = 'error'
    searchStatus.value = '❌ 检查失败:' + err
  }
}

async function rebuildIndex() {
  isRebuilding.value = true
  searchStatusType.value = ''
  searchStatus.value = '正在重建索引，请稍候...'
  try {
    await RebuildIndex()
    searchStatusType.value = 'success'
    searchStatus.value = '✅ 索引重建完成!'
  } catch (err) {
    searchStatusType.value = 'error'
    searchStatus.value = '❌ 索引重建失败:' + err
  } finally {
    isRebuilding.value = false
  }
}

async function testLLM() {
  llmTestStatusType.value = ''
  llmTestResult.value = '正在测试...'
  try {
    await SetSetting('llm_base_url', form.value.llm_base_url)
    await SetSetting('llm_model', form.value.llm_model)
    await SetSetting('llm_api_key', form.value.llm_api_key)
  } catch (e) {
    console.error('Save failed:', e)
  }
  try {
    const result = await TestLLMConnection()
    if (result.success) {
      llmTestStatusType.value = 'success'
      llmTestResult.value = '✅ ' + (result.message || 'LLM 连接成功')
    } else {
      llmTestStatusType.value = 'error'
      llmTestResult.value = '❌ ' + (result.message || 'LLM 连接失败')
    }
  } catch (err) {
    llmTestStatusType.value = 'error'
    llmTestResult.value = '❌ 测试失败:' + err
  }
}
</script>

<template>
  <div class="settings-container">
    <h2 class="page-title">设置</h2>

    <!-- 搜索引擎状态 -->
    <div class="section">
      <h3 class="section-title">搜索引擎</h3>
      <p class="help-text">BM25 + 向量 (Qdrant) + RRF 融合。重建索引会将所有笔记重新分块、嵌入向量。</p>
      <div class="button-group">
        <button @click="checkSearch" class="btn-secondary">检查状态</button>
        <button @click="rebuildIndex" :disabled="isRebuilding" class="btn-primary">
          {{ isRebuilding ? '⏳ 重建中...' : '🔄 重建索引' }}
        </button>
      </div>
      <div v-if="searchStatus" class="status-message" :class="searchStatusType">
        {{ searchStatus }}
      </div>
    </div>

    <!-- LLM 配置 -->
    <div class="section">
      <h3 class="section-title">LLM 问答配置</h3>
      <p class="help-text tips">💡 推荐 LM Studio 或 Ollama，加载聊天模型后填入地址</p>

      <div class="form-group">
        <label>API Base URL *</label>
        <input v-model="form.llm_base_url" placeholder="http://127.0.0.1:1234/v1 (LM Studio) 或 http://localhost:11434/v1 (Ollama)" class="input-field" />
      </div>

      <div class="form-row">
        <div class="form-group flex-1">
          <label>模型名称 *</label>
          <input v-model="form.llm_model" placeholder="qwen2.5, llama3, gpt-4" class="input-field" />
        </div>
        <div class="form-group flex-1">
          <label>API Key (可选)</label>
          <input v-model="form.llm_api_key" type="password" placeholder="本地部署通常不需要" class="input-field" />
        </div>
      </div>

      <button @click="testLLM" class="btn-secondary">🔌 测试 LLM 连接</button>
      <div v-if="llmTestResult" class="status-message" :class="llmTestStatusType">
        {{ llmTestResult }}
      </div>
    </div>

    <!-- Embedding 配置 -->
    <div class="section">
      <h3 class="section-title">Embedding 配置</h3>
      <p class="help-text">用于笔记向量化，和问答 LLM 独立。推荐在 LM Studio 中加载 embedding 模型。</p>

      <div class="form-group">
        <label>Embedding API Base URL *</label>
        <input v-model="form.embed_base_url" placeholder="http://127.0.0.1:1234/v1" class="input-field" />
      </div>

      <div class="form-row">
        <div class="form-group flex-1">
          <label>Embedding 模型名称 *</label>
          <input v-model="form.embed_model" placeholder="nomic-embed-text, text-embedding-ada-002" class="input-field" />
        </div>
        <div class="form-group flex-1">
          <label>API Key (可选)</label>
          <input v-model="form.embed_api_key" type="password" placeholder="通常本地不需要" class="input-field" />
        </div>
      </div>
    </div>

    <!-- AI 问答配置 -->
    <div class="section">
      <h3 class="section-title">AI 问答配置</h3>

      <div class="form-group">
        <label>搜索数量 (Top K)</label>
        <input v-model="form.ai_top_k" placeholder="5" type="number" class="input-field" style="width: 100px;" />
      </div>

      <div class="form-group">
        <label>系统提示词</label>
        <textarea v-model="form.ai_system_prompt" rows="4" class="input-field textarea"></textarea>
      </div>
    </div>

    <!-- 保存按钮 -->
    <div class="save-section">
      <button @click="saveSettings" class="btn-primary btn-large">💾 保存所有设置</button>
    </div>
  </div>
</template>

<style scoped>
.settings-container {
  padding: 2rem;
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
  overflow-y: auto;
  height: 100%;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0 0 2rem 0;
  color: #f9fafb;
}

.section {
  background-color: #1f2937;
  padding: 1.5rem;
  border-radius: 0.5rem;
  margin-bottom: 1.5rem;
  border: 1px solid #374151;
}

.section-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #14b8a6;
  margin: 0 0 1rem 0;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  font-size: 0.875rem;
  color: #9ca3af;
  margin-bottom: 0.5rem;
}

.input-field {
  width: 100%;
  padding: 0.625rem 0.875rem;
  background-color: #111827;
  border: 1px solid #374151;
  border-radius: 0.375rem;
  color: #f9fafb;
  font-size: 0.875rem;
  box-sizing: border-box;
}

.input-field:focus {
  outline: none;
  border-color: #0d9488;
}

.textarea {
  resize: vertical;
  min-height: 100px;
}

.help-text {
  font-size: 0.75rem;
  color: #10b981;
  margin-top: 0.5rem;
  margin-bottom: 1rem;
  padding: 0.5rem;
  background-color: #064e3b;
  border-radius: 0.25rem;
}

.tips {
  color: #fbbf24 !important;
  background-color: #78350f !important;
}

.form-row {
  display: flex;
  gap: 1rem;
}

.flex-1 {
  flex: 1;
}

.button-group {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
}

.status-message {
  margin-top: 1rem;
  padding: 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
}

.status-message.success {
  background-color: #064e3b;
  color: #6ee7b7;
  border: 1px solid #059669;
}

.status-message.error {
  background-color: #450a0a;
  color: #fca5a5;
  border: 1px solid #dc2626;
}

.save-section {
  display: flex;
  justify-content: flex-end;
  margin-top: 2rem;
}

.btn-primary,
.btn-secondary {
  padding: 0.625rem 1.25rem;
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

.btn-primary:hover:not(:disabled) {
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

.btn-large {
  padding: 0.75rem 2rem;
  font-size: 1rem;
}
</style>
