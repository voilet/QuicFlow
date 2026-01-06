<template>
  <div class="code-editor-wrapper" :style="{ height: height, width: '100%' }">
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
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import * as monaco from 'monaco-editor'

// 配置 Monaco Editor Web Worker
if (typeof window !== 'undefined' && !window.MonacoEnvironment) {
  // 使用 getWorker 方法创建 worker，避免 CORS 问题
  // 创建一个代理 worker，避免 postMessage 错误
  window.MonacoEnvironment = {
    getWorker: function (moduleId, label) {
      // 创建一个简单的代理 worker，响应所有消息但不实际处理
      // Monaco Editor 会在主线程运行实际逻辑，这样可以避免 CORS 问题
      const workerCode = `
        self.onmessage = function(e) {
          // 代理 worker，接收并响应消息，但实际处理在主线程
          // 这样可以避免 postMessage 错误
          try {
            if (e.data) {
              // 响应消息，避免错误
              self.postMessage({ 
                $type: 'response',
                id: e.data.id || null,
                success: true
              })
            }
          } catch (err) {
            // 忽略错误
          }
        };
      `
      const blob = new Blob([workerCode], { type: 'application/javascript' })
      const workerUrl = URL.createObjectURL(blob)
      return new Worker(workerUrl)
    }
  }
}

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

onMounted(async () => {
  // 等待 DOM 渲染完成
  await nextTick()
  
  let retryCount = 0
  const maxRetries = 10 // 最大重试次数，避免无限循环
  let initTimer = null
  
  // 对于对话框中的情况，需要更长的等待时间
  // 使用 MutationObserver 或多次重试确保容器已完全渲染
  const initEditor = async () => {
    // 清除之前的定时器，避免重复执行
    if (initTimer) {
      clearTimeout(initTimer)
      initTimer = null
    }
    
    retryCount++
    
    if (retryCount > maxRetries) {
      console.warn('CodeEditor: Max retries reached, giving up initialization')
      return
    }
    
    if (!editorContainer.value) {
      // 如果容器还不存在，稍后重试
      initTimer = setTimeout(initEditor, 100)
      return
    }

    // 检查容器是否有尺寸
    const rect = editorContainer.value.getBoundingClientRect()
    
    // 确保容器有宽度（高度可能为0如果还没渲染）
    if (rect.width === 0) {
      // 强制设置宽度
      editorContainer.value.style.width = '100%'
      editorContainer.value.style.minWidth = '0'
      editorContainer.value.style.boxSizing = 'border-box'
    }
    
    // 对于对话框中的编辑器，如果高度为0但宽度不为0，也可以尝试初始化
    // Monaco Editor 的 automaticLayout 会在容器有尺寸后自动调整
    if (rect.width === 0) {
      // 如果容器还没有宽度，稍后重试
      initTimer = setTimeout(initEditor, 100)
      return
    }

    try {
      // 如果编辑器已存在，先销毁
      if (editor) {
        editor.dispose()
        editor = null
      }

      // 确保容器样式正确
      if (editorContainer.value) {
        editorContainer.value.style.width = '100%'
        editorContainer.value.style.height = '100%'
        editorContainer.value.style.minWidth = '0'
        editorContainer.value.style.minHeight = '0'
        editorContainer.value.style.boxSizing = 'border-box'
      }

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
        },
        // 禁用某些需要 worker 的功能，避免错误
        quickSuggestions: false,
        suggestOnTriggerCharacters: false,
        acceptSuggestionOnEnter: 'off'
      })

      editor.onDidChangeModelContent(() => {
        emit('update:modelValue', editor.getValue())
      })

      // 确保编辑器正确布局（只尝试一次，automaticLayout 会自动处理）
      if (editor && editorContainer.value) {
        const rect = editorContainer.value.getBoundingClientRect()
        if (rect.width > 0 && rect.height > 0) {
          editor.layout()
        } else {
          // 如果高度为0，等待一下再布局（对话框可能还在动画中）
          setTimeout(() => {
            if (editor && editorContainer.value) {
              const newRect = editorContainer.value.getBoundingClientRect()
              if (newRect.width > 0 && newRect.height > 0) {
                editor.layout()
              }
            }
          }, 100)
        }
      }
    } catch (error) {
      console.error('CodeEditor: Failed to create editor', error)
      // 如果创建失败，稍后重试
      if (retryCount < maxRetries) {
        initTimer = setTimeout(initEditor, 200)
      }
    }
  }

  // 开始初始化（使用 requestAnimationFrame 确保在下一帧执行）
  requestAnimationFrame(() => {
    initEditor()
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

// 监听高度变化，重新布局编辑器
watch(() => props.height, () => {
  if (editor) {
    setTimeout(() => {
      editor.layout()
    }, 50)
  }
})
</script>

<style scoped>
.code-editor-wrapper {
  position: relative;
  border: 1px solid var(--tech-border, #dcdfe6);
  border-radius: 4px;
  overflow: hidden;
  background-color: #1e1e1e;
  width: 100%;
  min-width: 0;
  box-sizing: border-box;
}

.code-editor {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: block;
  box-sizing: border-box;
}

/* 确保 Monaco Editor 正确显示 */
.code-editor-wrapper :deep(.monaco-editor) {
  background-color: #1e1e1e !important;
}

.code-editor-wrapper :deep(.monaco-editor .margin) {
  background-color: #1e1e1e !important;
}

/* 修复 Monaco Editor 内部元素的宽度问题 */
.code-editor-wrapper :deep(.monaco-editor) {
  width: 100% !important;
}

.code-editor-wrapper :deep(.monaco-editor .monaco-editor-background) {
  width: 100% !important;
}

.code-editor-wrapper :deep(.monaco-editor .monaco-scrollable-element) {
  width: 100% !important;
  overflow: visible !important;
}

/* 修复 glyph-margin（行号边距）宽度 */
.code-editor-wrapper :deep(.monaco-editor .glyph-margin) {
  width: auto !important;
  min-width: 0 !important;
  max-width: none !important;
}

/* 修复 margin-view-zones（边距视图区域）宽度 */
.code-editor-wrapper :deep(.monaco-editor .margin-view-zones) {
  width: auto !important;
  min-width: 0 !important;
  max-width: none !important;
}

/* 修复 margin-view-overlays（边距视图覆盖层）宽度 */
.code-editor-wrapper :deep(.monaco-editor .margin-view-overlays) {
  width: auto !important;
  min-width: 0 !important;
  max-width: none !important;
}

/* 修复 ime-text-area（输入法文本区域）宽度 */
.code-editor-wrapper :deep(.monaco-editor .monaco-inputbox) {
  width: 100% !important;
}

.code-editor-wrapper :deep(.monaco-editor .monaco-inputbox .ime-text-area) {
  width: 100% !important;
  max-width: 100% !important;
  box-sizing: border-box !important;
}

/* 确保编辑器内容区域正确显示 */
.code-editor-wrapper :deep(.monaco-editor .monaco-scrollable-element > .monaco-editor-background) {
  width: 100% !important;
}

.code-editor-wrapper :deep(.monaco-editor .monaco-scrollable-element > .monaco-scrollable-element > .monaco-editor-background) {
  width: 100% !important;
}

/* 修复编辑器整体布局 */
.code-editor-wrapper :deep(.monaco-editor .overflow-guard) {
  width: 100% !important;
}

.code-editor-wrapper :deep(.monaco-editor .monaco-scrollable-element > .monaco-scrollable-element) {
  width: 100% !important;
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
