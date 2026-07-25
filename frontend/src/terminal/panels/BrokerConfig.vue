<script setup lang="ts">
import { ref } from 'vue'
import { useWailsApp } from '@/lib/composables/useWailsApp'
import { PanelHeader } from '@/terminal/components/panel'
import PanelShell from '@/terminal/components/panel/PanelShell.vue'

const state = ref<'loading' | 'loaded' | 'error' | 'empty'>('loaded')

defineProps<{ panelId: string; params?: Record<string, any> }>()

const broker = ref<'binance' | 'futu'>('binance')
const binanceKey = ref('')
const binanceSecret = ref('')
const binanceTestnet = ref(true)
const futu主机 = ref('localhost')
const futu端口 = ref(11111)

async function testConnection() {
  try {
    const app = useWailsApp()
    if (!app?.TestBrokerConnection) return
    const result = await app.TestBrokerConnection(broker.value, getCurrentConfig())
  } catch (e) { console.warn('TestConnection:', e) }
}

async function saveConfig() {
  try {
    const app = useWailsApp()
    if (!app?.SaveCredential) return
    await app.SaveCredential(`broker_${broker.value}`, broker.value, getCurrentConfig())
  } catch (e) { console.warn('SaveConfig:', e) }
}

function getCurrentConfig(): Record<string, string> {
  switch (broker.value) {
    case 'binance': return { api_key: binanceKey.value, secret_key: binanceSecret.value, testnet: String(binanceTestnet.value) }
    case 'futu': return { host: futu主机.value, port: String(futu端口.value) }
    default: return {}
  }
}
</script>

<template>
  <PanelShell :state="state">
    <template #loaded>
      <div class="broker-config-panel">
        <PanelHeader :title="$t('broker.title')" />
        <div class="broker-form">
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
          <div class="actions"><button class="btn" @click="testConnection">{{ $t('common.test') }}</button><button class="btn btn-primary" @click="saveConfig">{{ $t('common.save') }}</button></div>
        </div>
      </div>
    </template>
  </PanelShell>
</template>

<style scoped>
.broker-config-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.broker-form { flex: 1; overflow-y: auto; padding: var(--space-md) var(--panel-padding); }
.form-group { margin-bottom: var(--space-md); }
.form-group label { display: block; font-size: var(--font-xs); color: var(--color-text-tertiary); text-transform: uppercase; margin-bottom: var(--space-xs); }
.form-input { width: 100%; padding: var(--space-xs) var(--space-sm); background: var(--color-bg-input); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text-primary); font-size: var(--font-sm); outline: none; box-sizing: border-box; }
.form-input:focus { border-color: var(--color-accent); }
.checkbox-group label { font-size: var(--font-xs); color: var(--color-text-primary); display: flex; align-items: center; gap: var(--space-sm); text-transform: none; }
.config-section { margin-top: var(--space-md); }
.config-section .section-title { display: block; margin-bottom: var(--space-sm); }
.section-note { font-size: var(--font-xs); color: var(--color-accent); margin-bottom: var(--space-sm); }
.connection-status { display: flex; align-items: center; gap: var(--space-sm); margin-top: var(--space-sm); }
.status-dot { width: 8px; height: 8px; border-radius: 50%; }
.status-dot.on { background: var(--color-success); }
.status-dot.off { background: var(--color-text-tertiary); }
.status-text { font-size: var(--font-xs); color: var(--color-text-tertiary); }
.actions { display: flex; gap: var(--space-sm); margin-top: var(--space-lg); }
.actions .btn { flex: 1; }
</style>
