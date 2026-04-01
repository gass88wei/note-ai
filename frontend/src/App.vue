<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useRouter } from 'vue-router'
import { useNoteStore } from '@/stores'
import { CheckLeannStatus } from '../wailsjs/go/main/App'

const route = useRoute()
const router = useRouter()
const noteStore = useNoteStore()
const searchReady = ref(false)

onMounted(async () => {
  try {
    await noteStore.loadNotes()
    await noteStore.loadCategories()
  } catch (err) {
    console.error('加载失败:', err)
  }

  // Poll search engine status every 2s until ready
  const checkStatus = async () => {
    try {
      const status = await CheckLeannStatus()
      searchReady.value = status.available
    } catch {
      searchReady.value = false
    }
    if (!searchReady.value) {
      setTimeout(checkStatus, 2000)
    }
  }
  // Delay first check to let Qdrant start
  setTimeout(checkStatus, 3000)
})

const navItems = [
  { path: '/', label: '笔记', icon: 'note' },
  { path: '/chat', label: 'AI 对话', icon: 'chat' },
  { path: '/settings', label: '设置', icon: 'settings' },
]
</script>

<template>
  <div class="app-container">
    <!-- 侧边栏 -->
    <aside class="sidebar">
      <nav class="nav-menu">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: route.path === item.path }"
        >
          <span class="icon">
            <svg v-if="item.icon === 'note'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
            </svg>
            <svg v-else-if="item.icon === 'chat'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/>
            </svg>
            <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
              <path d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
            </svg>
          </span>
          <span class="label">{{ item.label }}</span>
        </router-link>
      </nav>

      <div class="sidebar-status">
        <span class="status-dot" :class="searchReady ? 'ready' : 'loading'"></span>
        <span class="status-text">{{ searchReady ? '就绪' : '加载中' }}</span>
      </div>
    </aside>

    <!-- 主内容区 -->
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.app-container {
  display: flex;
  height: 100vh;
  width: 100vw;
  background-color: #111827;
  color: #f9fafb;
}

.sidebar {
  width: 80px;
  background-color: #1f2937;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 1rem 0;
  border-right: 1px solid #374151;
}

.nav-menu {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0.75rem 1rem;
  border-radius: 0.5rem;
  text-decoration: none;
  color: #9ca3af;
  transition: all 0.2s;
  min-width: 70px;
  gap: 0.25rem;
}

.nav-item:hover {
  background-color: #374151;
  color: #f9fafb;
}

.nav-item.active {
  background-color: #0d9488;
  color: white;
}

.icon {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.icon svg {
  width: 100%;
  height: 100%;
}

.label {
  font-size: 0.75rem;
  font-weight: 500;
}

.sidebar-status {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  padding-bottom: 0.5rem;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  transition: background-color 0.3s;
}

.status-dot.ready {
  background-color: #10b981;
}

.status-dot.loading {
  background-color: #fbbf24;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.status-text {
  font-size: 0.625rem;
  color: #6b7280;
}

.main-content {
  flex: 1;
  overflow: hidden;
  position: relative;
}
</style>
