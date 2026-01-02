<template>
  <div class="code-editor-wrapper" :style="{ height: height }">
    <div ref="editorContainer" class="code-editor"></div>
    <div
      v-if="showPlaceholder && placeholder"
      class="code-editor-placeholder"
      @click="focusEditor"
    >
      {{ placeholderText }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import * as monaco from 'monaco-editor'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  language: {
    type: String,
    default: 'shell'
  },
  theme: {
    type: String,
    default: 'vs-dark'
  },
  height: {
    type: String,
    default: '200px'
  },
  readOnly: {
    type: Boolean,
    default: false
  },
  placeholder: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:modelValue'])

const editorContainer = ref(null)
let editor = null

// 计算是否显示 placeholder
const showPlaceholder = computed(() => {
  return !props.modelValue && !props.readOnly
})

// 处理 placeholder 中的 &#10; 换行符
const placeholderText = computed(() => {
  return props.placeholder.replace(/&#10;/g, '\n')
})

const focusEditor = () => {
  if (editor) {
    editor.focus()
  }
}

onMounted(() => {
  editor = monaco.editor.create(editorContainer.value, {
    value: props.modelValue || '',
    language: props.language,
    theme: props.theme,
    readOnly: props.readOnly,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    automaticLayout: true,
    lineNumbers: 'on',
    wordWrap: 'on',
    fontSize: 13,
    tabSize: 2,
    padding: { top: 8 },
    scrollbar: {
      vertical: 'auto',
      horizontal: 'auto'
    }
  })

  editor.onDidChangeModelContent(() => {
    emit('update:modelValue', editor.getValue())
  })
})

onBeforeUnmount(() => {
  if (editor) {
    editor.dispose()
  }
})

watch(() => props.modelValue, (newValue) => {
  if (editor && newValue !== editor.getValue()) {
    editor.setValue(newValue || '')
  }
})

watch(() => props.language, (newValue) => {
  if (editor) {
    monaco.editor.setModelLanguage(editor.getModel(), newValue)
  }
})

watch(() => props.theme, (newValue) => {
  if (editor) {
    monaco.editor.setTheme(newValue)
  }
})

watch(() => props.readOnly, (newValue) => {
  if (editor) {
    editor.updateOptions({ readOnly: newValue })
  }
})
</script>

<style scoped>
.code-editor-wrapper {
  position: relative;
  border: 1px solid var(--tech-border, #dcdfe6);
  border-radius: 4px;
  overflow: hidden;
}

.code-editor {
  width: 100%;
  height: 100%;
}

.code-editor-placeholder {
  position: absolute;
  top: 8px;
  left: 64px;
  right: 14px;
  color: #6b7280;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  pointer-events: none;
  user-select: none;
  opacity: 0.6;
}

.code-editor-placeholder:hover {
  cursor: text;
  pointer-events: auto;
}
</style>
