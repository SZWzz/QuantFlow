<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useThemeStore } from '@/lib/theme'
import { useSettingsStore } from '@/stores/settings'
import { setLocale } from '@/lib/i18n'
import { getIcon } from '@/lib/icons'
import { APP_VERSION } from '@/version'
import { saveCredential, getCredential, alertDialog } from '@/lib/wails'
import { logger } from '@/lib/logger'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()
const themeStore = useThemeStore()
const settingsStore = useSettingsStore()

const activeSection = ref('appearance')

interface Section {
  id: string
  label: string
  icon: string
}

const sections: Section[] = [
  { id: 'appearance', label: 'appearance', icon: getIcon('config') },
  { id: 'language', label: 'language', icon: getIcon('terminal') },
  { id: 'notifications', label: 'notifications', icon: getIcon('notify') },
  { id: 'data', label: 'data', icon: getIcon('quote') },
  { id: 'api', label: 'api', icon: getIcon('broker') },
  { id: 'trading', label: 'trading', icon: getIcon('order') },
  { id: 'display', label: 'display', icon: getIcon('market') },
  { id: 'shortcuts', label: 'shortcuts', icon: getIcon('command') },
  { id: 'storage', label: 'storage', icon: getIcon('portfolio') },
  { id: 'about', label: 'about', icon: getIcon('info') },
]

const dataSources = ['auto', 'yahoo', 'eastmoney', 'binance']
const dateFormats = ['YYYY-MM-DD', 'MM/DD/YYYY', 'DD/MM/YYYY']
const decimalOptions = [0, 2, 4]
const brokerOptions = ['paper', 'futu', 'longbridge', 'ibkr', 'binance', 'okx']
const saveMsg = ref('')

const apiKeys = ref({
  fred: '',
  finnhub: '',
  iwencai: '',
})

async function loadApiKeys() {
  for (const name of ['fred', 'finnhub', 'iwencai']) {
    const cred = await getCredential(`${name}_api_key`)
    if (cred?.api_key) {
      apiKeys.value[name as keyof typeof apiKeys.value] = cred.api_key
    }
  }
}

async function onSaveApiKeys() {
  try {
    for (const [name, key] of Object.entries(apiKeys.value)) {
      if (key) {
        await saveCredential(`${name}_api_key`, { api_key: key })
        logger.info(`[Settings] saved credential: ${name}`)
      }
    }
    saveMsg.value = '已保存到加密存储'
    setTimeout(() => saveMsg.value = '', 3000)
  } catch (e) {
    logger.error('[Settings] save api keys failed:', e)
    await alertDialog('保存 API 密钥失败')
  }
}

onMounted(() => { loadApiKeys() })

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
        <span class="nav-icon" v-html="sec.icon" />
        <span class="nav-label">{{ t(`settings.${sec.label}`) }}</span>
      </button>
    </nav>

    <div class="settings-content">
      <!-- Appearance -->
      <section v-if="activeSection === 'appearance'" class="section">
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('config')" />
          {{ t('settings.appearance') }}
        </h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.theme') }}</label>
          <div class="btn-group">
            <button
              :class="['option-btn', { active: themeStore.theme === 'dark' }]"
              @click="themeStore.setTheme('dark')"
            >
              <span class="opt-icon" v-html="getIcon('terminal')" />
              {{ t('settings.dark') }}
            </button>
            <button
              :class="['option-btn', { active: themeStore.theme === 'light' }]"
              @click="themeStore.setTheme('light')"
            >
              <span class="opt-icon" v-html="getIcon('market')" />
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
          <label class="form-label">{{ t('settings.color_scheme') }}</label>
          <div class="btn-group">
            <button :class="['option-btn', { active: settingsStore.settings.colorScheme === 'cn' }]"
              @click="settingsStore.update('colorScheme', 'cn'); themeStore.applyColorScheme('cn')">{{ t('settings.cn_colors') }}</button>
            <button :class="['option-btn', { active: settingsStore.settings.colorScheme === 'us' }]"
              @click="settingsStore.update('colorScheme', 'us'); themeStore.applyColorScheme('us')">{{ t('settings.us_colors') }}</button>
          </div>
        </div>
      </section>

      <!-- Language -->
      <section v-if="activeSection === 'language'" class="section">
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('terminal')" />
          {{ t('settings.language') }}
        </h3>

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
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('notify')" />
          {{ t('settings.notifications') }}
        </h3>

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
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('quote')" />
          {{ t('settings.data') }}
        </h3>

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
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('broker')" />
          {{ t('settings.api_keys') }}
        </h3>
        <p class="form-hint" style="margin-bottom: 14px">配置第三方数据源 API 密钥，密钥经 AES-256-GCM 加密后存储于系统凭据管理器。</p>

        <div class="form-group">
          <label class="form-label">{{ t('settings.fred_key') }} <span class="api-source">(美联储经济数据)</span></label>
          <input type="password" class="form-input"
            v-model="apiKeys.fred"
            placeholder="从 https://fred.stlouisfed.org/docs/api/api_key.html 申请" />
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.finnhub_key') }} <span class="api-source">(美股行情)</span></label>
          <input type="password" class="form-input"
            v-model="apiKeys.finnhub"
            placeholder="从 https://finnhub.io/register 免费注册" />
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.iwencai_key') }} <span class="api-source">(研报/公告搜索)</span></label>
          <input type="password" class="form-input"
            v-model="apiKeys.iwencai"
            placeholder="从 https://www.iwencai.com/ 申请" />
        </div>

        <div class="form-group">
          <button class="action-btn" @click="onSaveApiKeys">
            <span class="btn-icon" v-html="getIcon('save')" />
            保存 API 密钥
          </button>
          <span v-if="saveMsg" class="save-msg">{{ saveMsg }}</span>
        </div>
      </section>

      <!-- Trading -->
      <section v-if="activeSection === 'trading'" class="section">
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('order')" />
          {{ t('settings.trading') }}
        </h3>

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
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('market')" />
          {{ t('settings.display') }}
        </h3>

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
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('command')" />
          {{ t('settings.shortcuts') }}
        </h3>

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
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('portfolio')" />
          {{ t('settings.storage') }}
        </h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.db_path') }}</label>
          <input type="text" class="form-input" readonly value="~/.quantflow/quantflow.db" />
        </div>

        <div class="form-group">
          <button class="action-btn" @click="onExportData">
            <span class="btn-icon" v-html="getIcon('export')" />
            {{ t('settings.export_data') }}
          </button>
        </div>
      </section>

      <!-- About -->
      <section v-if="activeSection === 'about'" class="section">
        <h3 class="section-title">
          <span class="section-icon" v-html="getIcon('info')" />
          {{ t('settings.about') }}
        </h3>

        <div class="form-group">
          <label class="form-label">{{ t('settings.version') }}</label>
          <span class="form-value">{{ APP_VERSION }}</span>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('settings.license') }}</label>
          <span class="form-value">AGPL-3.0</span>
        </div>

        <div class="form-group">
          <label class="form-label">GitHub</label>
          <div class="link-list">
            <a href="https://github.com/quantflow/quantflow" target="_blank" rel="noopener" class="ext-link">
              <span class="link-icon" v-html="getIcon('terminal')" />
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
  background: var(--color-bg-panel);
}

/* Left nav */
.settings-nav {
  width: 160px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 8px 6px;
  border-right: 1px solid var(--color-border);
  overflow-y: auto;
  background: var(--color-bg-subtle);
  background-image: var(--gradient-card);
}

.nav-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: none;
  border: none;
  border-left: 3px solid transparent;
  color: var(--color-text-tertiary);
  font-size: 12px;
  text-align: left;
  cursor: pointer;
  transition: all var(--transition-fast);
  border-radius: 0 var(--radius-md) var(--radius-md) 0;
  position: relative;
}

.nav-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
}

.nav-btn.active {
  background: var(--color-accent-soft);
  color: var(--color-accent);
  border-left-color: var(--color-accent);
  box-shadow: 0 0 8px var(--color-accent-glow);
}

.nav-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  opacity: 0.6;
  transition: opacity var(--transition-fast);
}

.nav-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.nav-btn.active .nav-icon {
  opacity: 1;
}

.nav-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Right content */
.settings-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px;
}

.section {
  max-width: 520px;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(6px); }
  to { opacity: 1; transform: translateY(0); }
}

.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 18px 0;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--color-border);
}

.section-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  color: var(--color-accent);
}

.section-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

/* Form elements */
.form-group {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-bottom: 8px;
  font-weight: 500;
}

.form-input {
  width: 100%;
  padding: 8px 12px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  font-size: 12px;
  outline: none;
  transition: all var(--transition-fast);
  box-sizing: border-box;
  font-family: inherit;
}

.form-input:focus {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-soft);
}

.form-input::placeholder {
  color: var(--color-text-tertiary);
}

.form-input[readonly] {
  color: var(--color-text-tertiary);
  cursor: default;
}

.form-input-sm {
  width: 100px;
}

.form-select {
  width: 100%;
  padding: 8px 12px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-primary);
  font-size: 12px;
  outline: none;
  cursor: pointer;
  transition: all var(--transition-fast);
  box-sizing: border-box;
  font-family: inherit;
}

.form-select:focus {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-soft);
}

.form-range {
  width: 100%;
  accent-color: var(--color-accent);
  margin-top: 4px;
}

.form-value {
  font-size: 13px;
  color: var(--color-text-primary);
  font-weight: 500;
}

.form-hint {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 6px;
  line-height: 1.5;
}

/* Button groups */
.btn-group {
  display: flex;
  gap: 6px;
}

.option-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 6px 14px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text-tertiary);
  font-size: 12px;
  cursor: pointer;
  transition: all var(--transition-fast);
  font-family: inherit;
}

.option-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  border-color: var(--color-border-strong);
}

.option-btn.active {
  background: var(--color-accent-soft);
  color: var(--color-accent);
  border-color: var(--color-accent);
  box-shadow: 0 0 8px var(--color-accent-glow);
  font-weight: 500;
}

.opt-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 12px;
  height: 12px;
}

.opt-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 18px;
  background: var(--color-accent-soft);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-md);
  color: var(--color-accent);
  font-size: 12px;
  cursor: pointer;
  transition: all var(--transition-fast);
  font-family: inherit;
  font-weight: 500;
}

.action-btn:hover {
  background: var(--color-accent);
  color: var(--color-text-inverse);
  box-shadow: 0 0 12px var(--color-accent-glow);
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 13px;
  height: 13px;
}

.btn-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

/* Shortcut key */
.shortcut-key {
  font-size: 13px;
  color: var(--color-text-primary);
}

.shortcut-key kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 3px 10px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-family: inherit;
  font-size: 12px;
  color: var(--color-text-secondary);
  box-shadow: 0 1px 2px rgba(0,0,0,0.2);
}

/* Links */
.link-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ext-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-accent);
  text-decoration: none;
  transition: all var(--transition-fast);
  padding: 4px 0;
}

.ext-link:hover {
  text-decoration: underline;
  color: var(--color-accent-hover);
}

.link-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 13px;
  height: 13px;
}

.link-icon :deep(svg) {
  width: 100%;
  height: 100%;
}

.api-source { color: var(--color-text-tertiary); font-size: 10px; font-weight: normal; }
.save-msg { color: var(--color-success); font-size: 12px; margin-left: 10px; font-weight: 500; }
</style>
