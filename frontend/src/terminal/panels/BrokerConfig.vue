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
    <div class="form-group"><label>{{ $t('broker.title') }}</label><select v-model="broker" class="form-input"><option value="binance">Binance</option><option value="futu">Futu</option></select></div>
    <div v-if="broker === 'binance'" class="config-section">
      <h4 class="section-title">{{ $t('broker.config_title') }}</h4>
      <div class="form-group"><label>{{ $t('broker.api_key') }}</label><input v-model="binanceKey" type="password" class="form-input" :placeholder="$t('broker.api_key')" /></div>
      <div class="form-group"><label>{{ $t('broker.secret_key') }}</label><input v-model="binanceSecret" type="password" class="form-input" :placeholder="$t('broker.secret_key')" /></div>
      <div class="form-group checkbox-group"><label><input v-model="binanceTestnet" type="checkbox" /> Use Testnet (testnet.binance.vision)</label></div>
    </div>
    <div v-if="broker === 'futu'" class="config-section">
      <h4 class="section-title">{{ $t('broker.config_title') }}</h4>
      <p class="section-note">{{ $t('broker.futu_setup_hint') }}</p>
      <div class="form-group"><label>{{ $t('broker.host') }}</label><input v-model="futu主机" class="form-input" /></div>
      <div class="form-group"><label>{{ $t('broker.port') }}</label><input v-model.number="futu端口" type="number" class="form-input" /></div>
      <div class="connection-status"><span class="status-dot off"></span><span class="status-text">{{ $t('common.disconnected') }}</span></div>
    </div>
    <div class="actions"><button class="test-btn" @click="testConnection">{{ $t('common.test') }}</button><button class="save-btn" @click="saveConfig">{{ $t('common.save') }}</button></div>
  </div>
</template>

<style scoped>
.broker-config-panel { padding: 12px; background: var(--color-bg-panel); height: 100%; overflow-y: auto; }
.form-group { margin-bottom: 10px; }
.form-group label { display: block; font-size: 10px; color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: 4px; }
.form-input { width: 100%; padding: 6px 8px; background: var(--color-bg-input); border: 1px solid var(--color-accent-soft); border-radius: 4px; color: var(--color-text-primary); font-size: 13px; outline: none; box-sizing: border-box; }
.form-input:focus { border-color: #58a6ff; }
.checkbox-group label { font-size: 12px; color: var(--color-text-primary); display: flex; align-items: center; gap: 6px; text-transform: none; }
.config-section { margin-top: 12px; }
.section-title { font-size: 13px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 8px; }
.section-note { font-size: 11px; color: #f0883e; margin-bottom: 8px; }
.connection-status { display: flex; align-items: center; gap: 6px; margin-top: 8px; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; }
.status-dot.on { background: #3fb950; } .status-dot.off { background: var(--color-text-tertiary); }
.status-text { font-size: 11px; color: var(--color-text-tertiary); }
.actions { display: flex; gap: 8px; margin-top: 16px; }
.test-btn { flex: 1; padding: 8px; background: var(--color-accent-soft); border: none; border-radius: 4px; color: #58a6ff; font-size: 13px; font-weight: 600; cursor: pointer; }
.save-btn { flex: 1; padding: 8px; background: #0a3d1a; border: none; border-radius: 4px; color: #3fb950; font-size: 13px; font-weight: 600; cursor: pointer; }
</style>
