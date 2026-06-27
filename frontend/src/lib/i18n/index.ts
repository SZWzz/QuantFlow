import { createI18n } from 'vue-i18n'
import zh from './zh'
import en from './en'

type Locale = 'zh' | 'en'

const LOCALE_KEY = 'qf-locale'

const saved = localStorage.getItem(LOCALE_KEY)
const locale: Locale = saved === 'en' ? 'en' : 'zh'

export const i18n = createI18n({
  legacy: false,
  locale,
  fallbackLocale: 'en',
  messages: { zh, en },
})

export function setLocale(lang: Locale) {
  i18n.global.locale.value = lang
  localStorage.setItem(LOCALE_KEY, lang)
}
