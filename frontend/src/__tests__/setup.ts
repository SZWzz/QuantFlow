import { createI18n } from 'vue-i18n'
import { config } from '@vue/test-utils'

const i18n = createI18n({
  locale: 'zh',
  fallbackLocale: 'zh',
  messages: { zh: {}, en: {} },
  legacy: false,
})

config.global.plugins = [i18n]
