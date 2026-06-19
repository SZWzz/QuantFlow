import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHashHistory } from 'vue-router'
import { i18n } from './lib/i18n'
import App from './App.vue'

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
