<script setup lang="ts">
import { computed, ref } from 'vue'
import { highlightedGo } from '../highlight'

const props = defineProps<{ code: string }>()
const emit = defineEmits<{ (e: 'copied'): void }>()

const html = computed(() => highlightedGo(props.code))
const copied = ref(false)

async function copyCode() {
  try {
    await navigator.clipboard.writeText(props.code)
  } catch {
    // http 非 localhost 环境降级方案
    const ta = document.createElement('textarea')
    ta.value = props.code
    document.body.appendChild(ta)
    ta.select()
    document.execCommand('copy')
    document.body.removeChild(ta)
  }
  copied.value = true
  emit('copied')
  setTimeout(() => (copied.value = false), 1200)
}
</script>

<template>
  <n-card title="对应 Go 代码" size="small">
    <template #header-extra>
      <n-button size="small" type="primary" ghost @click="copyCode">
        {{ copied ? '已复制 ✓' : '复制代码' }}
      </n-button>
    </template>
    <pre class="code-pre"><code class="hljs" v-html="html"></code></pre>
  </n-card>
</template>
