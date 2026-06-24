<script setup lang="ts">
import { ref, nextTick, onMounted, onUnmounted, watch } from 'vue'
import { marked } from 'marked'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'

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
renderer.code = function (code: string, language: string | undefined) {
  const lang = language || ''
  if (lang && hljs.getLanguage(lang)) {
    return '<pre><code class="hljs ' + lang + '">' + hljs.highlight(code, { language: lang }).value + '</code></pre>'
  }
  return '<pre><code class="hljs">' + hljs.highlightAuto(code).value + '</code></pre>'
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
    if ((window as any).go?.main?.App) {
      const app = (window as any).go.main.App
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
  for (let i = 0; i < text.length; i += 3) {
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
    <div class="chat-header">
      <select v-model="selectedProfile" class="header-select">
        <option v-for="p in profiles" :key="p.name" :value="p.name">{{ p.display }}</option>
      </select>
      <select v-model="selectedModel" class="header-select model-select">
        <option v-for="m in availableModels" :key="m" :value="m">{{ m }}</option>
      </select>
      <button class="new-chat-btn" @click="newChat" title="新对话">+</button>
    </div>

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
      <button class="send-btn" @click="send" :disabled="isLoading">发送</button>
    </div>
  </div>
</template>

<style scoped>
.chat-panel { display: flex; flex-direction: column; height: 100%; background: #0d1117; }
.chat-header { display: flex; gap: 4px; padding: 6px 8px; border-bottom: 1px solid #21262d; background: #161b22; }
.header-select { flex: 1; padding: 4px 6px; background: #0d1117; border: 1px solid #30363d; border-radius: 4px; color: #c9d1d9; font-size: 11px; outline: none; max-width: 50%; }
.header-select:focus { border-color: #58a6ff; }
.new-chat-btn { padding: 4px 10px; background: #21262d; border: 1px solid #30363d; color: #c9d1d9; border-radius: 4px; cursor: pointer; font-size: 14px; }
.messages { flex: 1; overflow-y: auto; padding: 10px; display: flex; flex-direction: column; gap: 10px; }
.msg { max-width: 88%; padding: 10px 12px; border-radius: 8px; font-size: 12px; line-height: 1.6; }
.msg.user { align-self: flex-end; background: #1a3a5c; border: 1px solid #1a4a7c; }
.msg.assistant { align-self: flex-start; background: #161b22; border: 1px solid #30363d; }
.msg.system { align-self: center; background: #1a2332; border: 1px solid #30363d; max-width: 95%; font-size: 11px; color: #8b949e; }
.msg-role { font-size: 10px; color: #58a6ff; font-weight: 600; margin-bottom: 4px; display: flex; justify-content: space-between; align-items: center; }
.msg-time { font-weight: 400; color: #484f58; font-size: 9px; }
.msg-content :deep(h1), .msg-content :deep(h2), .msg-content :deep(h3) { color: #e6edf3; margin: 10px 0 6px; font-size: 15px; }
.msg-content :deep(p) { margin: 4px 0; color: #c9d1d9; }
.msg-content :deep(ul), .msg-content :deep(ol) { margin: 4px 0; padding-left: 18px; }
.msg-content :deep(li) { margin: 2px 0; color: #c9d1d9; }
.msg-content :deep(code) { background: #1a2332; padding: 2px 5px; border-radius: 3px; font-family: 'SF Mono', 'Cascadia Code', monospace; font-size: 11px; color: #58a6ff; }
.msg-content :deep(pre) { background: #0d1117; border: 1px solid #30363d; border-radius: 6px; padding: 10px; overflow-x: auto; margin: 6px 0; }
.msg-content :deep(pre code) { background: none; padding: 0; color: #c9d1d9; }
.msg-content :deep(table) { border-collapse: collapse; margin: 6px 0; width: 100%; font-size: 11px; }
.msg-content :deep(th) { background: #1a2332; padding: 4px 8px; text-align: left; border: 1px solid #30363d; color: #8b949e; }
.msg-content :deep(td) { padding: 3px 8px; border: 1px solid #30363d; color: #c9d1d9; }
.msg-content :deep(blockquote) { border-left: 3px solid #58a6ff; padding-left: 10px; margin: 6px 0; color: #8b949e; }
.msg-content :deep(strong) { color: #e6edf3; }
.tool-calls { margin-top: 8px; }
.tool-call-card { background: #0d1117; border: 1px solid #30363d; border-radius: 6px; margin-bottom: 4px; overflow: hidden; }
.tool-call-header { display: flex; align-items: center; gap: 6px; padding: 6px 8px; cursor: pointer; font-size: 11px; }
.tool-call-header:hover { background: #1a2332; }
.tool-call-icon { font-size: 8px; color: #8b949e; }
.tool-call-name { color: #bc8cff; font-weight: 500; }
.tool-call-body { padding: 6px 8px; border-top: 1px solid #21262d; }
.tool-section { margin-bottom: 6px; }
.tool-label { font-size: 9px; color: #484f58; text-transform: uppercase; display: block; margin-bottom: 2px; }
.tool-pre { font-size: 10px; color: #8b949e; white-space: pre-wrap; word-break: break-all; margin: 0; font-family: 'SF Mono', monospace; }
.token-info { font-size: 9px; color: #484f58; margin-top: 4px; text-align: right; }
.typing-indicator { display: flex; gap: 3px; }
.typing-indicator span { width: 6px; height: 6px; background: #484f58; border-radius: 50%; animation: typing 1.4s infinite; }
.typing-indicator span:nth-child(2) { animation-delay: 0.2s; }
.typing-indicator span:nth-child(3) { animation-delay: 0.4s; }
@keyframes typing { 0%, 60%, 100% { transform: translateY(0); opacity: 0.4; } 30% { transform: translateY(-6px); opacity: 1; } }
.input-area { display: flex; gap: 6px; padding: 8px; border-top: 1px solid #21262d; }
.chat-input { flex: 1; padding: 8px 10px; background: #161b22; border: 1px solid #30363d; border-radius: 6px; color: #c9d1d9; font-size: 12px; outline: none; }
.chat-input:focus { border-color: #58a6ff; }
.chat-input:disabled { opacity: 0.5; }
.chat-input::placeholder { color: #484f58; }
.send-btn { padding: 8px 16px; background: #1a3a5c; border: 1px solid #1a4a7c; color: #58a6ff; border-radius: 6px; cursor: pointer; font-size: 12px; font-weight: 600; }
.send-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.send-btn:hover:not(:disabled) { background: #1a4a7c; }
</style>
