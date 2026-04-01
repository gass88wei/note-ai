<script setup lang="ts">
import { ref, computed } from 'vue'
import { marked } from 'marked'
import type { Note } from '@/types'
import { useNoteStore } from '@/stores'

const props = defineProps<{ note: Note }>()
const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'save', data: { id: number; title: string; content: string; category: string; tags: string }): void
  (e: 'delete', id: number): void
}>()

const noteStore = useNoteStore()

const form = ref({
  title: props.note.title,
  content: props.note.content,
  category: props.note.category,
  tags: props.note.tags,
})

const categories = computed(() => noteStore.categories)
const isPreview = ref(false)

const renderedContent = computed(() => {
  return marked(form.value.content)
})

const save = () => {
  emit('save', {
    id: props.note.id,
    title: form.value.title,
    content: form.value.content,
    category: form.value.category,
    tags: form.value.tags,
  })
}

const cancel = () => {
  emit('cancel')
}

const deleteNote = () => {
  emit('delete', props.note.id)
}
</script>

<template>
  <div class="p-6">
    <div class="mb-4">
      <input
        v-model="form.title"
        placeholder="笔记标题"
        class="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-white placeholder-gray-400 focus:outline-none focus:border-primary-500"
      />
    </div>

    <div class="grid grid-cols-2 gap-4 mb-4">
      <div>
        <label class="block text-sm text-gray-400 mb-1">分类</label>
        <input
          v-model="form.category"
          list="category-list"
          class="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-white placeholder-gray-400 focus:outline-none focus:border-primary-500"
        />
        <datalist id="category-list">
          <option v-for="cat in categories" :key="cat" :value="cat" />
        </datalist>
      </div>
      <div>
        <label class="block text-sm text-gray-400 mb-1">标签 (逗号分隔)</label>
        <input
          v-model="form.tags"
          placeholder="tag1, tag2"
          class="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-white placeholder-gray-400 focus:outline-none focus:border-primary-500"
        />
      </div>
    </div>

    <div class="mb-2">
      <div class="flex items-center space-x-2 mb-2">
        <button
          @click="isPreview = false"
          class="px-3 py-1 text-sm rounded"
          :class="!isPreview ? 'bg-primary-600 text-white' : 'bg-gray-700 text-gray-300 hover:text-white'"
        >
          编辑
        </button>
        <button
          @click="isPreview = true"
          class="px-3 py-1 text-sm rounded"
          :class="isPreview ? 'bg-primary-600 text-white' : 'bg-gray-700 text-gray-300 hover:text-white'"
        >
          预览
        </button>
      </div>

      <div v-if="!isPreview">
        <textarea
          v-model="form.content"
          rows="30"
          placeholder="在这里写笔记 (支持 Markdown)..."
          class="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-white placeholder-gray-400 focus:outline-none focus:border-primary-500 resize-none font-mono text-sm"
        ></textarea>
      </div>
      <div v-else class="p-4 bg-gray-800 rounded h-96 overflow-auto">
        <div class="prose prose-invert max-w-none" v-html="renderedContent"></div>
      </div>
    </div>

    <div class="flex justify-end space-x-2">
      <button
        @click="deleteNote"
        class="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition-colors"
      >
        删除
      </button>
      <button
        @click="cancel"
        class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded transition-colors"
      >
        取消
      </button>
      <button
        @click="save"
        class="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors"
      >
        保存
      </button>
    </div>
  </div>
</template>
