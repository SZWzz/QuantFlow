<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const broker = ref<'binance' | 'futu'>('binance')
const binanceKey = ref('')
const binanceSecret = ref('')
const binanceTestnet = ref(true)
const futu主机 = ref('localhost')
const futu端口 = ref(11111)

function testConnection() { alert('Connection test: not yet wired to Go backend') }
function saveConfig() { alert('Config saved: not yet wired to Go backend') }
</script>

<template>
  <div class="broker-config-panel">
    <div class="form-group"><label>券商</label><select v-model="broker" class="form-input"><option value="binance">Binance</option><option value="futu">Futu</option></select></div>
    <div v-if="broker === 'binance'" class="config-section">
      <h4 class="section-title">币安 API 配置</h4>
      <div class="form-group"><label>API Key</label><input v-model="binanceKey" type="password" class="form-input" placeholder="Enter API Key" /></div>
      <div class="form-group"><label>Secret Key</label><input v-model="binanceSecret" type="password" class="form-input" placeholder="Enter Secret Key" /></div>
      <div class="form-group checkbox-group"><label><input v-model="binanceTestnet" type="checkbox" /> Use Testnet (testnet.binance.vision)</label></div>
    </div>
    <div v-if="broker === 'futu'" class="config-section">
      <h4 class="section-title">富途 OpenD 连接</h4>
      <p class="section-note">连接前请先本地启动 FutuOpenD</p>
      <div class="form-group"><label>主机</label><input v-model="futu主机" class="form-input" /></div>
      <div class="form-group"><label>端口</label><input v-model.number="futu端口" type="number" class="form-input" /></div>
      <div class="connection-status"><span class="status-dot off"></span><span class="status-text">未连接</span></div>
    </div>
    <div class="actions"><button class="test-btn" @click="testConnection">测试连接</button><button class="save-btn" @click="saveConfig">保存</button></div>
  </div>
</template>

<style scoped>
.broker-config-panel { padding: 12px; background: #1a1a2e; height: 100%; overflow-y: auto; }
.form-group { margin-bottom: 10px; }
.form-group label { display: block; font-size: 10px; color: #5a6380; text-transform: uppercase; margin-bottom: 4px; }
.form-input { width: 100%; padding: 6px 8px; background: #0f2137; border: 1px solid #1a3a5c; border-radius: 4px; color: #c9d1d9; font-size: 13px; outline: none; box-sizing: border-box; }
.form-input:focus { border-color: #58a6ff; }
.checkbox-group label { font-size: 12px; color: #c9d1d9; display: flex; align-items: center; gap: 6px; text-transform: none; }
.config-section { margin-top: 12px; }
.section-title { font-size: 13px; font-weight: 600; color: #e0e0e0; margin-bottom: 8px; }
.section-note { font-size: 11px; color: #f0883e; margin-bottom: 8px; }
.connection-status { display: flex; align-items: center; gap: 6px; margin-top: 8px; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; }
.status-dot.on { background: #3fb950; } .status-dot.off { background: #5a6380; }
.status-text { font-size: 11px; color: #5a6380; }
.actions { display: flex; gap: 8px; margin-top: 16px; }
.test-btn { flex: 1; padding: 8px; background: #1a3a5c; border: none; border-radius: 4px; color: #58a6ff; font-size: 13px; font-weight: 600; cursor: pointer; }
.save-btn { flex: 1; padding: 8px; background: #0a3d1a; border: none; border-radius: 4px; color: #3fb950; font-size: 13px; font-weight: 600; cursor: pointer; }
</style>
