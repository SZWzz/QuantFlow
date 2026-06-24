import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHashHistory } from 'vue-router'
import { i18n } from './lib/i18n'
import App from './App.vue'

// Wails v3 runtime bridge — creates window.go shim using @wailsio/runtime.
// This allows existing Wails v2-style (window as any).go.main.App.XXX calls
// to work transparently with Wails v3's Call.ByName API.
import { setupWailsBridge } from './lib/wails'
setupWailsBridge()

// Global design tokens — must load before component styles
import './assets/themes.css'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'terminal',
      component: () => import('@/terminal/TerminalMode.vue'),
    },
    {
      path: '/workflow',
      name: 'workflow',
      component: () => import('@/workflow/WorkflowMode.vue'),
    },
  ],
})

const pinia = createPinia()
const app = createApp(App)

app.use(router)
app.use(i18n)
app.use(pinia)
app.mount('#app')
