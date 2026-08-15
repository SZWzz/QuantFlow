<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  time: string
  toolCalls?: ToolCallEntry[]
  tokens?: { prompt: number; completion: number }
}

interface ToolCallEntry {
  tool: string
  args: string
  result: string
  expanded: boolean
}

interface AgentProfile {
  name: string
  display: string
}

// Configure marked. Note: the `highlight` option was removed in marked v18.
// Syntax highlighting is done via a custom renderer for code blocks instead.
marked.setOptions({
  breaks: true,
})

// Custom renderer: highlight code blocks with highlight.js
const renderer = new marked.Renderer()
renderer.code = function ({ text, lang }: { text: string; lang?: string }) {
  const language = lang || ''
  if (language && hljs.getLanguage(language)) {
    return '<pre><code class="hljs ' + language + '">' + hljs.highlight(text, { language: language }).value + '</code></pre>'
  }
  return '<pre><code class="hljs">' + hljs.highlightAuto(text).value + '</code></pre>'
}
marked.setOptions({ renderer })

const messages = ref<Message[]>([
  {
    id: 'welcome',
    role: 'assistant',
    content: '你好！我是 QuantFlow AI 助手。选择 Agent Profile 和模型后开始对话。',
    time: '--',
  },
])
const input = ref('')
const isLoading = ref(false)
const profiles = ref<AgentProfile[]>([
  { name: 'general', display: '通用助理' },
  { name: 'quant_analyst', display: '量化分析师' },
  { name: 'trader', display: '交易员' },
  { name: 'research_assistant', display: '研究助理' },
])
const selectedProfile = ref('general')
const selectedModel = ref('ollama/llama3.1:8b')
const availableModels = ref<string[]>([
  'ollama/llama3.1:8b',
  'openai/gpt-4o',
  'anthropic/claude-sonnet-4-6',
])
const messagesContainer = ref<HTMLElement | null>(null)
const pythonConnected = ref(false)
let _active = true
onUnmounted(() => { _active = false })

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

function renderMarkdown(text: string): string {
  try {
    return marked.parse(text) as string
  } catch {
    return text
  }
}

async function send() {
  const text = input.value.trim()
  if (!text || isLoading.value) return

  const userMsg: Message = {
    id: crypto.randomUUID(),
    role: 'user',
    content: text,
    time: new Date().toLocaleTimeString(),
  }
  messages.value.push(userMsg)
  input.value = ''
  isLoading.value = true

  const assistantId = crypto.randomUUID()
  const assistantMsg: Message = {
    id: assistantId,
    role: 'assistant',
    content: '',
    time: new Date().toLocaleTimeString(),
    toolCalls: [],
  }
  messages.value.push(assistantMsg)
  scrollToBottom()

  try {
    const app = useWailsApp()
    if (app) {
      const result = await app.Chat(selectedProfile.value, selectedModel.value, text)
      assistantMsg.content = result || 'No response.'
    } else {
      await simulateStreamingResponse(assistantMsg, text)
    }
  } catch (e: any) {
    assistantMsg.content = `Error: ${e.message || e}. Is the Python sidecar running?`
    pythonConnected.value = false
  }

  isLoading.value = false
  scrollToBottom()
}

async function simulateStreamingResponse(msg: Message, prompt: string) {
  const responses: Record<string, string> = {
    general: `I'm the QuantFlow 通用助理. You asked: "${prompt}". Here's a sample markdown response:\n\n## Analysis\n\n| Metric | Value |\n|--------|-------|\n| Sharpe | 1.42 |\n| Max DD | -8.7% |\n\n\`\`\`python\ndef backtest():\n    return {"sharpe": 1.42}\n\`\`\`\n\n**Note:** Connect to the Python sidecar for real AI responses.`,
    quant_analyst: `As a 量化分析师, let me analyze: "${prompt}".\n\n### Factor Analysis\n\n- **momentum_20d**: IC 0.035, IR 0.42\n- **rsi_14**: IC 0.018, IR 0.21\n\n> Recommendation: Use momentum factors for this strategy with at least 3-month holding period.`,
    trader: `Trade analysis for: "${prompt}".\n\n### Trade Setup\n- **Entry**: Wait for confirmation above resistance\n- **Stop Loss**: 2 ATR below entry (~3.5%)\n- **Target**: 2:1 R:R\n- **Position Size**: 2% risk per trade\n\nAlways manage risk first!`,
    research_assistant: `Research on: "${prompt}".\n\n## Key Findings\n\n1. **Industry**: Growing at 12% CAGR\n2. **Competitive Position**: Strong moat (brand + network effects)\n3. **Valuation**: P/E 22x vs industry 25x — slightly undervalued`,
  }
  const text = responses[selectedProfile.value] || responses.general
  for (let i = 0; i < text.length && _active; i += 3) {
    msg.content += text.slice(i, i + 3)
    await new Promise((r) => setTimeout(r, 20))
    scrollToBottom()
  }
}

function newChat() {
  messages.value = [
    {
      id: 'welcome',
      role: 'assistant',
      content: '新对话已开始。',
      time: new Date().toLocaleTimeString(),
    },
  ]
}

function toggleToolCall(msg: Message, idx: number) {
  if (msg.toolCalls && msg.toolCalls[idx]) {
    msg.toolCalls[idx].expanded = !msg.toolCalls[idx].expanded
  }
}

watch(() => messages.value.length, scrollToBottom)
</script>

<template>
  <div class="chat-panel">
    <!-- Header: Profile + Model selectors -->
    <PanelHeader title="AI 对话">
      <template #controls>
        <select v-model="selectedProfile" class="header-select">
          <option v-for="p in profiles" :key="p.name" :value="p.name">{{ p.display }}</option>
        </select>
        <select v-model="selectedModel" class="header-select">
          <option v-for="m in availableModels" :key="m" :value="m">{{ m }}</option>
        </select>
        <button class="btn btn-sm" @click="newChat" title="新对话">+</button>
      </template>
    </PanelHeader>

    <!-- Messages -->
    <div ref="messagesContainer" class="messages">
      <div v-for="msg in messages" :key="msg.id" :class="['msg', msg.role]">
        <div class="msg-role">
          {{ msg.role === 'user' ? 'You' : msg.role === 'system' ? 'System' : 'AI' }}
          <span class="msg-time">{{ msg.time }}</span>
        </div>
        <div class="msg-content" v-html="renderMarkdown(msg.content)"></div>

        <!-- Tool call cards -->
        <div v-if="msg.toolCalls && msg.toolCalls.length > 0" class="tool-calls">
          <div v-for="(tc, i) in msg.toolCalls" :key="i" class="tool-call-card">
            <div class="tool-call-header" @click="toggleToolCall(msg, i)">
              <span class="tool-call-icon">{{ tc.expanded ? '▼' : '▶' }}</span>
              <span class="tool-call-name">🔧 {{ tc.tool }}</span>
            </div>
            <div v-if="tc.expanded" class="tool-call-body">
              <div class="tool-section">
                <span class="tool-label">Args:</span>
                <pre class="tool-pre">{{ tc.args }}</pre>
              </div>
              <div class="tool-section">
                <span class="tool-label">Result:</span>
                <pre class="tool-pre">{{ tc.result }}</pre>
              </div>
            </div>
          </div>
        </div>

        <div v-if="msg.tokens" class="token-info">
          {{ msg.tokens.prompt }} + {{ msg.tokens.completion }} tokens
        </div>
      </div>

      <div v-if="isLoading" class="msg assistant">
        <div class="msg-content typing-indicator">
          <span></span><span></span><span></span>
        </div>
      </div>
    </div>

    <!-- Input area -->
    <div class="input-area">
      <input
        v-model="input"
        type="text"
        :placeholder="isLoading ? 'AI 思考中...' : '向 AI 助手提问...'"
        class="chat-input"
        :disabled="isLoading"
        @keyup.enter="send"
      />
      <button class="btn btn-sm btn-primary send-btn" @click="send" :disabled="isLoading">发送</button>
    </div>
  </div>
</template>

<style scoped>
.chat-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.header-select { min-width: 0; max-width: 45%; padding: var(--space-xs) var(--space-sm); background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-primary); font-size: var(--font-xs); outline: none; }
.header-select:focus { border-color: var(--color-accent); }
/* 聊天气泡/消息流为自绘布局（PanelTable 表达不了），保留但全部 token 化 */
.messages { flex: 1; overflow-y: auto; padding: var(--space-md); display: flex; flex-direction: column; gap: var(--space-md); }
.msg { max-width: 88%; padding: var(--space-md); border-radius: var(--radius-lg); font-size: var(--font-xs); line-height: 1.6; }
.msg.user { align-self: flex-end; background: var(--color-accent-soft); border: 1px solid var(--color-accent); }
.msg.assistant { align-self: flex-start; background: var(--color-bg-input); border: 1px solid var(--color-border); }
.msg.system { align-self: center; background: var(--color-bg-subtle); border: 1px solid var(--color-border); max-width: 95%; font-size: var(--font-xs); color: var(--color-text-tertiary); }
.msg-role { font-size: var(--font-xs); color: var(--color-accent); font-weight: 600; margin-bottom: var(--space-xs); display: flex; justify-content: space-between; align-items: center; }
.msg-time { font-weight: 400; color: var(--color-text-tertiary); font-size: var(--font-xs); }
.msg-content :deep(h1), .msg-content :deep(h2), .msg-content :deep(h3) { color: var(--color-text-primary); margin: var(--space-md) 0 var(--space-sm); font-size: var(--font-lg); }
.msg-content :deep(p) { margin: var(--space-xs) 0; color: var(--color-text-primary); }
.msg-content :deep(ul), .msg-content :deep(ol) { margin: var(--space-xs) 0; padding-left: var(--space-lg); }
.msg-content :deep(li) { margin: var(--space-xs) 0; color: var(--color-text-primary); }
.msg-content :deep(code) { background: var(--color-bg-subtle); padding: var(--space-xs); border-radius: var(--radius-sm); font-family: var(--font-mono); font-size: var(--font-xs); color: var(--color-accent); }
.msg-content :deep(pre) { background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-md); padding: var(--space-md); overflow-x: auto; margin: var(--space-sm) 0; }
.msg-content :deep(pre code) { background: none; padding: 0; color: var(--color-text-primary); }
.msg-content :deep(table) { border-collapse: collapse; margin: var(--space-sm) 0; width: 100%; font-size: var(--font-xs); }
.msg-content :deep(th) { background: var(--color-bg-subtle); padding: var(--space-xs) var(--space-sm); text-align: left; border: 1px solid var(--color-border); color: var(--color-text-tertiary); }
.msg-content :deep(td) { padding: var(--space-xs) var(--space-sm); border: 1px solid var(--color-border); color: var(--color-text-primary); }
.msg-content :deep(blockquote) { border-left: 3px solid var(--color-accent); padding-left: var(--space-md); margin: var(--space-sm) 0; color: var(--color-text-tertiary); }
.msg-content :deep(strong) { color: var(--color-text-primary); }
.tool-calls { margin-top: var(--space-sm); }
.tool-call-card { background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-md); margin-bottom: var(--space-xs); overflow: hidden; }
.tool-call-header { display: flex; align-items: center; gap: var(--space-sm); padding: var(--space-xs) var(--space-sm); cursor: pointer; font-size: var(--font-xs); }
.tool-call-header:hover { background: var(--color-bg-subtle); }
.tool-call-icon { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.tool-call-name { color: var(--color-accent); font-weight: 500; }
.tool-call-body { padding: var(--space-xs) var(--space-sm); border-top: 1px solid var(--color-border); }
.tool-section { margin-bottom: var(--space-sm); }
.tool-label { font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; display: block; margin-bottom: var(--space-xs); }
.tool-pre { font-size: var(--font-xs); color: var(--color-text-tertiary); white-space: pre-wrap; word-break: break-all; margin: 0; font-family: var(--font-mono); }
.token-info { font-size: var(--font-xs); color: var(--color-text-tertiary); margin-top: var(--space-xs); text-align: right; }
.typing-indicator { display: flex; gap: var(--space-xs); }
.typing-indicator span { width: 6px; height: 6px; background: var(--color-text-tertiary); border-radius: 50%; animation: typing 1.4s infinite; }
.typing-indicator span:nth-child(2) { animation-delay: 0.2s; }
.typing-indicator span:nth-child(3) { animation-delay: 0.4s; }
@keyframes typing { 0%, 60%, 100% { transform: translateY(0); opacity: 0.4; } 30% { transform: translateY(-6px); opacity: 1; } }
.input-area { display: flex; gap: var(--space-sm); padding: var(--space-sm) var(--panel-padding); border-top: 1px solid var(--color-border-subtle); flex-shrink: 0; }
.chat-input { flex: 1; padding: var(--space-sm) var(--space-md); background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-md); color: var(--color-text-primary); font-size: var(--font-xs); outline: none; }
.chat-input:focus { border-color: var(--color-accent); }
.chat-input:disabled { opacity: 0.5; }
.chat-input::placeholder { color: var(--color-text-tertiary); }
.send-btn { flex-shrink: 0; }
</style>
