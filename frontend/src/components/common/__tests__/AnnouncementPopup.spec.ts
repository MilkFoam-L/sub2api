import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AnnouncementPopup from '../AnnouncementPopup.vue'

const mockStore = vi.hoisted(() => ({
  currentPopup: {
    id: 1,
    title: '系统公告',
    content: '兼容一段很长的链接 https://example.com/path/with/a/very/very/very/long/segment/that/should/not-overflow-card-border',
    notify_mode: 'popup' as const,
    created_at: '2026-06-03T12:00:00Z',
    updated_at: '2026-06-03T12:00:00Z'
  },
  dismissPopup: vi.fn()
}))

vi.mock('@/stores/announcements', () => ({
  useAnnouncementStore: () => mockStore
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'announcements.unread': '未读公告',
      'announcements.markRead': '我知道了'
    }[key] ?? key)
  })
}))

vi.mock('@/utils/format', () => ({
  formatRelativeWithDateTime: () => '刚刚'
}))

const mountPopup = () => mount(AnnouncementPopup, {
  global: {
    stubs: {
      Teleport: { template: '<div><slot /></div>' }
    }
  }
})

describe('AnnouncementPopup', () => {
  beforeEach(() => {
    mockStore.dismissPopup.mockClear()
    mockStore.currentPopup = {
      id: 1,
      title: '系统公告',
      content: '兼容一段很长的链接 https://example.com/path/with/a/very/very/very/long/segment/that/should/not-overflow-card-border',
      notify_mode: 'popup',
      created_at: '2026-06-03T12:00:00Z',
      updated_at: '2026-06-03T12:00:00Z'
    }
  })

  it('centers the popup and uses theme tokens', () => {
    const wrapper = mountPopup()
    const overlay = wrapper.get('[data-test="announcement-popup-overlay"]')
    const panel = wrapper.get('[data-test="announcement-popup-panel"]')

    expect(overlay.classes()).toContain('items-center')
    expect(overlay.classes()).not.toContain('items-start')
    expect(panel.classes()).toContain('bg-card')
    expect(panel.classes()).toContain('border-border')
    expect(panel.classes()).toContain('overflow-hidden')
  })

  it('keeps long announcement content inside the panel border', () => {
    const wrapper = mountPopup()
    const content = wrapper.get('[data-test="announcement-popup-content"]')
    const markdown = wrapper.get('[data-test="announcement-popup-markdown"]')

    expect(content.classes()).toContain('min-w-0')
    expect(content.classes()).toContain('overflow-y-auto')
    expect(markdown.classes()).toContain('break-words')
    expect(markdown.classes()).toContain('[overflow-wrap:anywhere]')
  })
})
