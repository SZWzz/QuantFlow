<script setup lang="ts">
import { onMounted } from 'vue'
import { useNotifyStore } from '@/stores/notify'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const store = useNotifyStore()
const levelIcon: Record<string, string> = { trade: '💹', warn: '⚠️', error: '❌', info: 'ℹ️' }

onMounted(() => store.fetchNotifications())
</script>

<template>
  <div class="notify-panel">
    <div class="filter-bar">
      <button v-for="lvl in ['all','trade','warn','error','info']" :key="lvl"
        :class="['filter-btn', { active: store.levelFilter === lvl }]"
        @click="store.setFilter(lvl)">{{ $t('notify.' + lvl) }}</button>
      <span class="unread-badge" v-if="store.unreadCount > 0">{{ store.unreadCount }} {{ $t('notify.new') }}</span>
    </div>
    <div class="notify-list">
      <div v-for="n in store.filteredNotifications" :key="n.id"
        :class="['notify-row', { unread: !n.is_read }]"
        @click="store.markRead(n.id)">
        <span class="notify-icon">{{ levelIcon[n.level] || '📌' }}</span>
        <div class="notify-body">
          <span class="notify-title">{{ n.title }}</span>
          <span class="notify-text">{{ n.body }}</span>
        </div>
        <span class="notify-time">{{ n.created_at }}</span>
      </div>
      <div v-if="!store.filteredNotifications.length" class="empty-state">{{ $t('notify.no_notifications') }}</div>
    </div>
    <div class="footer">
      <button class="read-all-btn" @click="store.markAllRead">{{ $t('notify.mark_all_read') }}</button>
    </div>
  </div>
</template>

<style scoped>
.notify-panel { padding: 10px; background: var(--color-bg-panel); height: 100%; display: flex; flex-direction: column; }
.filter-bar { display: flex; gap: 4px; margin-bottom: 8px; align-items: center; }
.filter-btn { padding: 3px 10px; background: var(--color-bg-input); border: 1px solid var(--color-accent-soft); border-radius: var(--radius-sm); color: var(--color-text-tertiary); font-size: 10px; text-transform: capitalize; cursor: pointer; }
.filter-btn.active { background: var(--color-accent-soft); color: var(--color-accent); border-color: var(--color-accent); }
.unread-badge { margin-left: auto; padding: 2px 8px; background: var(--color-down); border-radius: var(--radius-lg); color: var(--color-down); font-size: 10px; font-weight: 600; }
.notify-list { flex: 1; overflow-y: auto; }
.notify-row { display: flex; gap: 8px; padding: 8px; border-bottom: 1px solid var(--color-bg-input); cursor: pointer; transition: background 0.15s; }
.notify-row:hover { background: var(--color-bg-subtle); }
.notify-row.unread { background: var(--color-bg-selected); }
.notify-icon { font-size: 16px; flex-shrink: 0; margin-top: 2px; }
.notify-body { flex: 1; min-width: 0; }
.notify-title { display: block; font-size: 12px; font-weight: 600; color: var(--color-text-primary); }
.notify-row.unread .notify-title { color: var(--color-text-primary); }
.notify-text { display: block; font-size: 11px; color: var(--color-text-tertiary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.notify-time { font-size: 10px; color: var(--color-text-tertiary); flex-shrink: 0; }
.empty-state { padding: 40px; text-align: center; color: var(--color-text-tertiary); font-size: 13px; }
.footer { padding-top: 8px; border-top: 1px solid var(--color-bg-input); }
.read-all-btn { width: 100%; padding: 6px; background: var(--color-bg-subtle); border: 1px solid var(--color-accent-soft); border-radius: var(--radius-sm); color: var(--color-accent); font-size: 12px; cursor: pointer; }
</style>
