import { createRouter, createWebHashHistory } from 'vue-router'
import NotesView from '@/views/NotesView.vue'
import ChatView from '@/views/ChatView.vue'
import SettingsView from '@/views/SettingsView.vue'

const routes = [
  {
    path: '/',
    name: 'notes',
    component: NotesView,
  },
  {
    path: '/chat',
    name: 'chat',
    component: ChatView,
  },
  {
    path: '/settings',
    name: 'settings',
    component: SettingsView,
  },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

export default router
