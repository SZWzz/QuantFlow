import { createI18n } from 'vue-i18n'
import { config } from '@vue/test-utils'
import zh from '@/lib/i18n/zh'
import en from '@/lib/i18n/en'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages: { zh, en },
})

config.global.plugins = [i18n]
