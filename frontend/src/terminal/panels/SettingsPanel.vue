<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useThemeStore } from '@/lib/theme'
import { useSettingsStore } from '@/stores/settings'
import { setLocale } from '@/lib/i18n'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()
const themeStore = useThemeStore()
const settingsStore = useSettingsStore()

const activeSection = ref('appearance')

interface Section {
  id: string
  label: string
}

const sections: Section[] = [
  { id: 'appearance', label: 'appearance' },
  { id: 'language', label: 'language' },
  { id: 'notifications', label: 'notifications' },
  { id: 'data', label: 'data' },
  { id: 'api', label: 'api' },
  { id: 'trading', label: 'trading' },
  { id: 'display', label: 'display' },
  { id: 'shortcuts', label: 'shortcuts' },
  { id: 'storage', label: 'storage' },
  { id: 'about', label: 'about' },
]

const dataSources = ['auto', 'yahoo', 'eastmoney', 'binance']
const dateFormats = ['YYYY-MM-DD', 'MM/DD/YYYY', 'DD/MM/YYYY']
const decimalOptions = [0, 2, 4]
const brokerOptions = ['paper', 'futu', 'longbridge', 'ibkr', 'binance', 'okx']
const saveMsg = ref('')

async function onSaveApiKeys() {
  const app = (window as any).go?.main?.App
  if (!app?.UpdateConfig) { saveMsg.value = '后端不可用'; return }
  try {
    await app.UpdateConfig({
      api_keys: {
        fred: settingsStore.settings.fredApiKey,
        finnhub: settingsStore.settings.finnhubApiKey,
        iwencai: settingsStore.settings.iwencaiApiKey,
      }
    })
    saveMsg.value = '已保存，重启后生效'
    setTimeout(() => saveMsg.value = '', 3000)
  } catch { saveMsg.value = '保存失败' }
}

function onExportData() {
  alert('Export data stub — not yet implemented')
}
</script>

<template>
  <div class="settings-panel">
    <nav class="settings-nav">
      <button
        v-for="sec in sections"
        :key="sec.id"
        :class="['nav-btn', { active: activeSection === sec.id }]"
        @click="activeSection = sec.id"
      >
        {{ t(`settings.${sec.label}`) }}
      </button>
    </nav>

    <div class="settings-content">
      <!-- Appearance -->
      <section v-if="activeSection === 'appearance'" class="section">
        <h3 class="section-title">{{ t('settings.appearance') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.theme') }}</label>
          <div class="btn-group">
            <button
              :class="['option-btn', { active: themeStore.theme === 'dark' }]"
              @click="themeStore.setTheme('dark')"
            >
              {{ t('settings.dark') }}
            </button>
            <button
              :class="['option-btn', { active: themeStore.theme === 'light' }]"
              @click="themeStore.setTheme('light')"
            >
              {{ t('settings.light') }}
            </button>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.density') }}</label>
          <div class="btn-group">
            <button
              v-for="d in (['compact', 'default', 'comfortable'] as const)"
              :key="d"
              :class="['option-btn', { active: themeStore.density === d }]"
              @click="themeStore.setDensity(d)"
            >
              {{ t(`settings.${d}`) }}
            </button>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">涨跌颜色</label>
          <div class="btn-group">
            <button :class="['option-btn', { active: settingsStore.settings.colorScheme === 'cn' }]"
              @click="settingsStore.update('colorScheme', 'cn'); themeStore.applyColorScheme('cn')">A股 (红涨绿跌)</button>
            <button :class="['option-btn', { active: settingsStore.settings.colorScheme === 'us' }]"
              @click="settingsStore.update('colorScheme', 'us'); themeStore.applyColorScheme('us')">美股 (绿涨红跌)</button>
          </div>
        </div>
      </section>

      <!-- Language -->
      <section v-if="activeSection === 'language'" class="section">
        <h3 class="section-title">{{ t('settings.language') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.language') }}</label>
          <select
            class="form-select"
            :value="settingsStore.settings.language"
            @change="(e) => { const v = (e.target as HTMLSelectElement).value; settingsStore.update('language', v); setLocale(v as 'zh' | 'en') }"
          >
            <option value="zh">{{ t('settings.chinese') }}</option>
            <option value="en">{{ t('settings.english') }}</option>
          </select>
        </div>
      </section>

      <!-- Notifications -->
      <section v-if="activeSection === 'notifications'" class="section">
        <h3 class="section-title">{{ t('settings.notifications') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.telegram_token') }}</label>
          <input
            type="text"
            class="form-input"
            :value="settingsStore.settings.telegramToken"
            placeholder="123456:ABC-DEF1234ghikl-zyx57W2v1u123ew11"
            @input="(e) => settingsStore.update('telegramToken', (e.target as HTMLInputElement).value)"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.chat_id') }}</label>
          <input
            type="text"
            class="form-input"
            :value="settingsStore.settings.telegramChatId"
            placeholder="-1001234567890"
            @input="(e) => settingsStore.update('telegramChatId', (e.target as HTMLInputElement).value)"
          />
        </div>
      </section>

      <!-- Data -->
      <section v-if="activeSection === 'data'" class="section">
        <h3 class="section-title">{{ t('settings.data') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.data_source') }}</label>
          <select
            class="form-select"
            :value="settingsStore.settings.dataSource"
            @change="(e) => settingsStore.update('dataSource', (e.target as HTMLSelectElement).value)"
          >
            <option v-for="ds in dataSources" :key="ds" :value="ds">{{ ds }}</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.cache_ttl') }}</label>
          <input
            type="number"
            class="form-input"
            :value="settingsStore.settings.cacheTtlDays"
            min="1"
            max="365"
            @input="(e) => settingsStore.update('cacheTtlDays', Number((e.target as HTMLInputElement).value))"
          />
        </div>
      </section>

      <!-- API Keys -->
      <section v-if="activeSection === 'api'" class="section">
        <h3 class="section-title">API 密钥</h3>
        <p class="form-hint" style="margin-bottom: 14px">配置第三方数据源 API 密钥，保存后写入 config.yaml 并在下次启动生效。</p>

        <div class="form-group">
          <label class="form-label">FRED API Key <span class="api-source">(美联储经济数据)</span></label>
          <input type="password" class="form-input"
            :value="settingsStore.settings.fredApiKey"
            placeholder="从 https://fred.stlouisfed.org/docs/api/api_key.html 申请"
            @input="(e) => settingsStore.update('fredApiKey', (e.target as HTMLInputElement).value)" />
        </div>

        <div class="form-group">
          <label class="form-label">Finnhub API Key <span class="api-source">(美股行情)</span></label>
          <input type="password" class="form-input"
            :value="settingsStore.settings.finnhubApiKey"
            placeholder="从 https://finnhub.io/register 免费注册"
            @input="(e) => settingsStore.update('finnhubApiKey', (e.target as HTMLInputElement).value)" />
        </div>

        <div class="form-group">
          <label class="form-label">爱问财 API Key <span class="api-source">(研报/公告搜索)</span></label>
          <input type="password" class="form-input"
            :value="settingsStore.settings.iwencaiApiKey"
            placeholder="从 https://www.iwencai.com/ 申请"
            @input="(e) => settingsStore.update('iwencaiApiKey', (e.target as HTMLInputElement).value)" />
        </div>

        <div class="form-group">
          <button class="action-btn" @click="onSaveApiKeys">
            保存到 config.yaml
          </button>
          <span v-if="saveMsg" class="save-msg">{{ saveMsg }}</span>
        </div>
      </section>

      <!-- Trading -->
      <section v-if="activeSection === 'trading'" class="section">
        <h3 class="section-title">{{ t('settings.trading') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.default_broker') }}</label>
          <select
            class="form-select"
            :value="settingsStore.settings.defaultBroker"
            @change="(e) => settingsStore.update('defaultBroker', (e.target as HTMLSelectElement).value)"
          >
            <option v-for="b in brokerOptions" :key="b" :value="b">{{ b }}</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.default_qty') }}</label>
          <input
            type="number"
            class="form-input"
            :value="settingsStore.settings.defaultQty"
            min="1"
            @input="(e) => settingsStore.update('defaultQty', Number((e.target as HTMLInputElement).value))"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.max_position') }}: {{ settingsStore.settings.maxPositionPct }}%</label>
          <input
            type="range"
            class="form-range"
            :value="settingsStore.settings.maxPositionPct"
            min="1"
            max="100"
            @input="(e) => settingsStore.update('maxPositionPct', Number((e.target as HTMLInputElement).value))"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.stop_loss') }}: {{ settingsStore.settings.stopLossPct }}%</label>
          <input
            type="range"
            class="form-range"
            :value="settingsStore.settings.stopLossPct"
            min="1"
            max="50"
            @input="(e) => settingsStore.update('stopLossPct', Number((e.target as HTMLInputElement).value))"
          />
        </div>
      </section>

      <!-- Display -->
      <section v-if="activeSection === 'display'" class="section">
        <h3 class="section-title">{{ t('settings.display') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.currency') }}</label>
          <input
            type="text"
            class="form-input form-input-sm"
            :value="settingsStore.settings.currency"
            maxlength="5"
            @input="(e) => settingsStore.update('currency', (e.target as HTMLInputElement).value)"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.decimals') }}</label>
          <select
            class="form-select"
            :value="settingsStore.settings.decimals"
            @change="(e) => settingsStore.update('decimals', Number((e.target as HTMLSelectElement).value))"
          >
            <option v-for="d in decimalOptions" :key="d" :value="d">{{ d }}</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.date_format') }}</label>
          <select
            class="form-select"
            :value="settingsStore.settings.dateFormat"
            @change="(e) => settingsStore.update('dateFormat', (e.target as HTMLSelectElement).value)"
          >
            <option v-for="df in dateFormats" :key="df" :value="df">{{ df }}</option>
          </select>
        </div>
      </section>

      <!-- Shortcuts -->
      <section v-if="activeSection === 'shortcuts'" class="section">
        <h3 class="section-title">{{ t('settings.shortcuts') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.toggle_mode') }}</label>
          <div class="shortcut-key">
            <kbd>Ctrl</kbd> + <kbd>W</kbd>
          </div>
          <p class="form-hint">{{ t('settings.toggle_mode') }}: Terminal / Workflow</p>
        </div>
      </section>

      <!-- Storage -->
      <section v-if="activeSection === 'storage'" class="section">
        <h3 class="section-title">{{ t('settings.storage') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.db_path') }}</label>
          <input type="text" class="form-input" readonly value="~/.quantflow/quantflow.db" />
        </div>

        <div class="form-group">
          <button class="action-btn" @click="onExportData">
            {{ t('settings.export_data') }}
          </button>
        </div>
      </section>

      <!-- About -->
      <section v-if="activeSection === 'about'" class="section">
        <h3 class="section-title">{{ t('settings.about') }}</h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.version') }}</label>
          <span class="form-value">2026.6.17</span>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.license') }}</label>
          <span class="form-value">AGPL-3.0</span>
        </div>

        <div class="form-group">
          <label class="form-label">GitHub</label>
          <div class="link-list">
            <a href="https://github.com/quantflow/quantflow" target="_blank" rel="noopener" class="ext-link">
              quantflow/quantflow
            </a>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.settings-panel {
  display: flex;
  height: 100%;
  background: #1a1a2e;
}

/* Left nav */
.settings-nav {
  width: 140px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 8px 0;
  border-right: 1px solid #0f2137;
  overflow-y: auto;
}

.nav-btn {
  padding: 8px 16px;
  background: none;
  border: none;
  border-left: 3px solid transparent;
  color: #5a6380;
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.nav-btn:hover {
  background: #16213e;
  color: #c0c8d8;
}

.nav-btn.active {
  background: #16213e;
  color: #58a6ff;
  border-left-color: #58a6ff;
}

/* Right content */
.settings-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}

.section {
  max-width: 480px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #e0e0e0;
  margin: 0 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid #0f2137;
}

/* Form elements */
.form-group {
  margin-bottom: 14px;
}

.form-label {
  display: block;
  font-size: 11px;
  color: #5a6380;
  margin-bottom: 6px;
}

.form-input {
  width: 100%;
  padding: 7px 10px;
  background: #0f2137;
  border: 1px solid #1a3a5c;
  border-radius: 4px;
  color: #e0e0e0;
  font-size: 12px;
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}

.form-input:focus {
  border-color: #58a6ff;
}

.form-input[readonly] {
  color: #5a6380;
  cursor: default;
}

.form-input-sm {
  width: 80px;
}

.form-select {
  width: 100%;
  padding: 7px 10px;
  background: #0f2137;
  border: 1px solid #1a3a5c;
  border-radius: 4px;
  color: #e0e0e0;
  font-size: 12px;
  outline: none;
  cursor: pointer;
  transition: border-color 0.15s;
  box-sizing: border-box;
}

.form-select:focus {
  border-color: #58a6ff;
}

.form-range {
  width: 100%;
  accent-color: #58a6ff;
  margin-top: 4px;
}

.form-value {
  font-size: 13px;
  color: #e0e0e0;
}

.form-hint {
  font-size: 10px;
  color: #5a6380;
  margin-top: 6px;
}

/* Button groups */
.btn-group {
  display: flex;
  gap: 4px;
}

.option-btn {
  padding: 5px 14px;
  background: #0f2137;
  border: 1px solid #1a3a5c;
  border-radius: 4px;
  color: #5a6380;
  font-size: 11px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.option-btn:hover {
  background: #16213e;
  color: #c0c8d8;
}

.option-btn.active {
  background: #1a3a5c;
  color: #58a6ff;
  border-color: #58a6ff;
}

.action-btn {
  padding: 7px 18px;
  background: #1a3a5c;
  border: 1px solid #2a5a8c;
  border-radius: 4px;
  color: #58a6ff;
  font-size: 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.action-btn:hover {
  background: #2a5a8c;
}

/* Shortcut key */
.shortcut-key {
  font-size: 13px;
  color: #e0e0e0;
}

.shortcut-key kbd {
  display: inline-block;
  padding: 2px 8px;
  background: #0f2137;
  border: 1px solid #1a3a5c;
  border-radius: 3px;
  font-family: inherit;
  font-size: 12px;
  color: #58a6ff;
}

/* Links */
.link-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ext-link {
  font-size: 12px;
  color: #58a6ff;
  text-decoration: none;
}

.ext-link:hover {
  text-decoration: underline;
}

.api-source { color: #5a6380; font-size: 10px; font-weight: normal; }
.save-msg { color: #22c55e; font-size: 12px; margin-left: 10px; }
</style>
