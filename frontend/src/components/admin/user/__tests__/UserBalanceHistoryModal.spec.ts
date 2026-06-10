import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserBalanceHistoryModal from '../UserBalanceHistoryModal.vue'
import type { AdminUser } from '@/types'

const { getUserBalanceHistory } = vi.hoisted(() => ({
  getUserBalanceHistory: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      getUserBalanceHistory,
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/utils/format', () => ({
  formatDateTime: (value: string) => value,
}))

const createUser = (overrides: Partial<AdminUser> = {}): AdminUser => ({
  id: 42,
  username: 'admin-target',
  email: 'target@example.com',
  role: 'user',
  balance: 10,
  concurrency: 1,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-04-17T00:00:00Z',
  updated_at: '2026-04-17T00:00:00Z',
  notes: '',
  current_concurrency: 0,
  ...overrides,
})

const mountModal = () => mount(UserBalanceHistoryModal, {
  props: {
    show: false,
    user: createUser(),
    refreshKey: 0,
  },
  global: {
    stubs: {
      BaseDialog: {
        props: ['show', 'title', 'width'],
        template: '<div v-if="show"><slot /></div>',
      },
      Select: {
        props: ['modelValue', 'options'],
        emits: ['update:modelValue', 'change'],
        template: '<select @change="$emit(\'change\')"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>',
      },
      Icon: true,
    },
  },
})

describe('UserBalanceHistoryModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getUserBalanceHistory.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 15,
      pages: 1,
      total_recharged: 0,
    })
  })

  it('reloads the first page when refreshKey changes while open', async () => {
    const wrapper = mountModal()

    await wrapper.setProps({ show: true })
    await flushPromises()
    expect(getUserBalanceHistory).toHaveBeenCalledWith(42, 1, 15, undefined)

    getUserBalanceHistory.mockClear()
    await wrapper.setProps({ refreshKey: 1 })
    await flushPromises()

    expect(getUserBalanceHistory).toHaveBeenCalledWith(42, 1, 15, undefined)
  })
})
