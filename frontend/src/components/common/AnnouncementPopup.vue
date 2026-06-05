<template>
  <Teleport to="body">
    <Transition name="popup-fade">
      <div
        v-if="announcementStore.currentPopup"
        data-test="announcement-popup-overlay"
        class="fixed inset-0 z-[120] flex items-center justify-center overflow-y-auto bg-background/75 p-4 backdrop-blur-xl"
      >
        <div
          data-test="announcement-popup-panel"
          class="relative flex max-h-[min(84vh,760px)] w-full max-w-[720px] min-w-0 flex-col overflow-hidden rounded-[2rem] border border-border bg-card text-foreground shadow-2xl shadow-primary/15 ring-1 ring-primary/15"
          @click.stop
        >
          <div class="pointer-events-none absolute inset-0 overflow-hidden rounded-[2rem]">
            <div class="absolute -right-12 -top-16 h-44 w-44 rounded-full bg-primary/20 blur-3xl"></div>
            <div class="absolute -bottom-20 -left-16 h-48 w-48 rounded-full bg-accent/30 blur-3xl"></div>
          </div>

          <header class="relative border-b border-border bg-card/90 px-6 py-5 sm:px-8">
            <div class="flex min-w-0 items-start justify-between gap-4">
              <div class="flex min-w-0 gap-4">
                <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl border border-primary/25 bg-primary/10 text-primary shadow-lg shadow-primary/10">
                  <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
                  </svg>
                </div>
                <div class="min-w-0">
                  <div class="mb-2 inline-flex items-center gap-2 rounded-full border border-primary/20 bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
                    <span class="relative flex h-2 w-2">
                      <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-primary opacity-60"></span>
                      <span class="relative inline-flex h-2 w-2 rounded-full bg-primary"></span>
                    </span>
                    {{ t('announcements.unread') }}
                  </div>
                  <h2 class="break-words text-xl font-semibold leading-tight text-foreground sm:text-2xl">
                    {{ announcementStore.currentPopup.title }}
                  </h2>
                  <div class="mt-2 flex min-w-0 items-center gap-1.5 text-sm text-muted-foreground">
                    <svg class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                      <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <time class="min-w-0 break-words">{{ formatRelativeWithDateTime(announcementStore.currentPopup.created_at) }}</time>
                  </div>
                </div>
              </div>
            </div>
          </header>

          <section
            data-test="announcement-popup-content"
            class="relative min-w-0 flex-1 overflow-y-auto bg-background/60 px-6 py-6 sm:px-8"
          >
            <div class="min-w-0 rounded-2xl border border-border bg-card/70 p-5 shadow-sm">
              <div
                data-test="announcement-popup-markdown"
                class="markdown-body prose prose-sm max-w-none break-words text-foreground prose-headings:text-foreground prose-p:text-foreground prose-a:text-primary prose-strong:text-foreground prose-code:break-all prose-code:text-primary dark:prose-invert [overflow-wrap:anywhere]"
                v-html="renderedContent"
              ></div>
            </div>
          </section>

          <footer class="relative border-t border-border bg-card/90 px-6 py-4 sm:px-8">
            <div class="flex justify-end">
              <button
                @click="handleDismiss"
                class="btn btn-primary px-5 py-2.5 text-sm shadow-lg shadow-primary/20 ring-1 ring-primary/25"
              >
                <span class="flex items-center gap-2">
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  {{ t('announcements.markRead') }}
                </span>
              </button>
            </div>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeWithDateTime } from '@/utils/format'

const { t } = useI18n()
const announcementStore = useAnnouncementStore()

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  const content = announcementStore.currentPopup?.content
  if (!content) return ''
  const html = marked.parse(content) as string
  return DOMPurify.sanitize(html)
})

function handleDismiss() {
  announcementStore.dismissPopup()
}

// Manage body overflow — only set, never unset (bell component handles restore)
watch(
  () => announcementStore.currentPopup,
  (popup) => {
    if (popup) {
      document.body.style.overflow = 'hidden'
    }
  }
)
</script>

<style scoped>
.popup-fade-enter-active {
  transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.popup-fade-leave-active {
  transition: all 0.2s cubic-bezier(0.4, 0, 1, 1);
}

.popup-fade-enter-from,
.popup-fade-leave-to {
  opacity: 0;
}

.popup-fade-enter-from > div {
  transform: scale(0.96) translateY(12px);
  opacity: 0;
}

.popup-fade-leave-to > div {
  transform: scale(0.98) translateY(8px);
  opacity: 0;
}

.overflow-y-auto::-webkit-scrollbar {
  width: 8px;
}

.overflow-y-auto::-webkit-scrollbar-track {
  background: transparent;
}

.overflow-y-auto::-webkit-scrollbar-thumb {
  background: hsl(var(--border));
  border-radius: 9999px;
}

.overflow-y-auto::-webkit-scrollbar-thumb:hover {
  background: hsl(var(--muted-foreground) / 0.45);
}
</style>
