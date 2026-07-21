<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotifyStore } from '@/stores/notify'
import { PanelHeader, EmptyState } from '@/terminal/components/panel'

defineProps<{ panelId: string; params?: Record<string, any> }>()

const { t } = useI18n()
const store = useNotifyStore()
const levelIcon: Record<string, string> = { trade: '💹', warn: '⚠️', error: '❌', info: 'ℹ️' }

const levelTabs = computed(() =>
  ['all', 'trade', 'warn', 'error', 'info'].map(lvl => ({ key: lvl, label: t('notify.' + lvl) }))
)

onMounted(() => store.fetchNotifications())
</script>

<template>
  <div class="notify-panel">
    <PanelHeader
      :title="$t('notify.title')"
      :tabs="levelTabs"
      :active-tab="store.levelFilter"
      @tab-change="store.setFilter"
    >
      <template #controls>
        <span v-if="store.unreadCount > 0" class="unread-badge">{{ store.unreadCount }} {{ $t('notify.new') }}</span>
      </template>
    </PanelHeader>

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
      <EmptyState v-if="!store.filteredNotifications.length" :title="$t('notify.no_notifications')" />
    </div>

    <div class="footer">
      <button class="btn btn-sm read-all-btn" @click="store.markAllRead">{{ $t('notify.mark_all_read') }}</button>
    </div>
  </div>
</template>

<style scoped>
.notify-panel { height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.unread-badge {
  padding: var(--space-xs) var(--space-sm); background: var(--color-down); border-radius: var(--radius-lg);
  color: var(--color-text-inverse); font-size: var(--font-xs); font-weight: 600;
}
.notify-list { flex: 1; overflow-y: auto; padding: 0 var(--panel-padding); }
/* 自绘通知行：图标 + 标题/正文双行 + 时间，PanelTable 表达不了，保留但 token 化 */
.notify-row {
  display: flex; gap: var(--space-sm); padding: var(--space-sm);
  border-bottom: 1px solid var(--color-border-subtle); cursor: pointer;
  transition: background var(--transition-fast);
}
.notify-row:hover { background: var(--color-bg-subtle); }
.notify-row.unread { background: var(--color-bg-selected); }
.notify-icon { font-size: var(--font-lg); flex-shrink: 0; margin-top: var(--space-xs); }
.notify-body { flex: 1; min-width: 0; }
.notify-title { display: block; font-size: var(--font-xs); font-weight: 600; color: var(--color-text-primary); }
.notify-text { display: block; font-size: var(--font-xs); color: var(--color-text-tertiary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.notify-time { font-size: var(--font-xs); color: var(--color-text-tertiary); flex-shrink: 0; }

.footer { padding: var(--space-sm) var(--panel-padding); border-top: 1px solid var(--color-border-subtle); flex-shrink: 0; }
.read-all-btn { width: 100%; color: var(--color-accent); border-color: var(--color-accent-soft); }
</style>
