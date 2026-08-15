/// <reference types="vitest" />
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'
import { readFileSync } from 'fs'

const pkg = JSON.parse(readFileSync(resolve(__dirname, 'package.json'), 'utf-8'))

const plugins: any = [vue()]
// Bundle analyzer: run `npm run build:analyze` to open treemap
if (process.env.ANALYZE) {
  try {
    const { visualizer } = require('rollup-plugin-visualizer') as any
    plugins.push(visualizer({ open: true, filename: 'dist/stats.html', gzipSize: true, brotliSize: true }))
  } catch { /* rollup-plugin-visualizer not installed — run: npm i -D rollup-plugin-visualizer */ }
}

export default defineConfig({
  plugins,
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2020',
    chunkSizeWarningLimit: 500,
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-vue': ['vue', 'vue-router', 'vue-i18n', 'pinia'],
          'vendor-flow': ['@vue-flow/core', '@vue-flow/background', '@vue-flow/controls', '@vue-flow/minimap'],
          'vendor-chart': ['echarts', 'vue-echarts'],
          'vendor-wails': ['@wailsio/runtime'],
          'vendor-markdown': ['marked', 'highlight.js'],
        },
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    exclude: ['e2e/**', 'node_modules/**', 'dist/**'],
    setupFiles: ['./vitest.setup.ts', './src/test-utils/setup.ts'],
    define: {
      __APP_VERSION__: JSON.stringify(pkg.version),
    },
    coverage: {
      thresholds: {
        lines: 40,
        branches: 30,
        functions: 40,
        statements: 40,
      },
    },
  },
})
