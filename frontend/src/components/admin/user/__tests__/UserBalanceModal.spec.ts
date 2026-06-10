import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserBalanceModal from '../UserBalanceModal.vue'
import type { AdminUser } from '@/types'

const { updateBalance } = vi.hoisted(() => ({
  updateBalance: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      updateBalance,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
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

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    name: 'BaseDialog',
    props: ['show', 'title', 'width'],
    template: '<div v-if="show"><slot /><slot name="footer" /></div>',
  },
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

describe('UserBalanceModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('closes and emits the updated user after a successful deposit', async () => {
    const updatedUser = createUser({ balance: 15 })
    updateBalance.mockResolvedValue(updatedUser)

    const wrapper = mount(UserBalanceModal, {
      props: {
        show: true,
        user: createUser(),
        operation: 'add',
      },
    })

    await wrapper.get('input[type="number"]').setValue('5')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateBalance).toHaveBeenCalledWith(42, 5, 'add', '')
    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(wrapper.emitted('success')).toEqual([[updatedUser]])
  })
})
