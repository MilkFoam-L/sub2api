import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfilePrivacyFilterCard from '@/components/user/profile/ProfilePrivacyFilterCard.vue'
import type { User } from '@/types'

const {
  updateProfileMock,
  showSuccessMock,
  showErrorMock,
  authStoreState
} = vi.hoisted(() => ({
  updateProfileMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  authStoreState: {
    user: null as User | null
  }
}))

vi.mock('@/api', () => ({
  userAPI: {
    updateProfile: updateProfileMock
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStoreState
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const messages: Record<string, string> = {
          'profile.privacyFilterTitle': 'Privacy Filter Protection',
          'profile.privacyFilterDescription': 'Redact sensitive data before forwarding requests.',
          'profile.privacyFilterEnabled': 'Protection enabled',
          'profile.privacyFilterDisabled': 'Protection disabled',
          'profile.privacyFilterEnabledHint': 'Sensitive text is redacted first.',
          'profile.privacyFilterDisabledHint': 'Requests continue with original content.',
          'profile.privacyFilterEnableSuccess': 'Privacy filter enabled',
          'profile.privacyFilterDisableSuccess': 'Privacy filter disabled',
          'profile.privacyFilterUpdateFailed': 'Failed to update privacy filter setting'
        }
        return messages[key] ?? key
      }
    })
  }
})

function createUser(overrides: Partial<User> = {}): User {
  return {
    id: 5,
    username: 'alice',
    email: 'alice@example.com',
    avatar_url: null,
    role: 'user',
    balance: 10,
    concurrency: 2,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    privacy_filter_enabled: false,
    created_at: '2026-04-20T00:00:00Z',
    updated_at: '2026-04-20T00:00:00Z',
    ...overrides
  }
}

function mountCard(user: User | null) {
  return mount(ProfilePrivacyFilterCard, {
    props: { user },
    global: {
      stubs: {
        Icon: true
      }
    }
  })
}

describe('ProfilePrivacyFilterCard', () => {
  beforeEach(() => {
    updateProfileMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
    authStoreState.user = null
  })

  it('renders the current privacy filter state', () => {
    const wrapper = mountCard(createUser({ privacy_filter_enabled: true }))

    expect(wrapper.text()).toContain('Privacy Filter Protection')
    expect(wrapper.text()).toContain('Protection enabled')
    expect(wrapper.get('[data-testid="profile-privacy-filter-toggle"]').attributes('aria-checked')).toBe('true')
  })

  it('updates the user preference when toggled on', async () => {
    const initialUser = createUser({ privacy_filter_enabled: false })
    const updatedUser = createUser({ privacy_filter_enabled: true })
    authStoreState.user = initialUser
    updateProfileMock.mockResolvedValue(updatedUser)

    const wrapper = mountCard(initialUser)
    await wrapper.get('[data-testid="profile-privacy-filter-toggle"]').trigger('click')
    await flushPromises()

    expect(updateProfileMock).toHaveBeenCalledWith({
      privacy_filter_enabled: true
    })
    expect(authStoreState.user).toBe(updatedUser)
    expect(showSuccessMock).toHaveBeenCalledWith('Privacy filter enabled')
    expect(showErrorMock).not.toHaveBeenCalled()
  })

  it('rolls back the switch and shows an error when the update fails', async () => {
    const initialUser = createUser({ privacy_filter_enabled: false })
    authStoreState.user = initialUser
    updateProfileMock.mockRejectedValue({ message: 'save failed' })

    const wrapper = mountCard(initialUser)
    await wrapper.get('[data-testid="profile-privacy-filter-toggle"]').trigger('click')
    await flushPromises()

    expect(updateProfileMock).toHaveBeenCalledWith({
      privacy_filter_enabled: true
    })
    expect(wrapper.get('[data-testid="profile-privacy-filter-toggle"]').attributes('aria-checked')).toBe('false')
    expect(showErrorMock).toHaveBeenCalledWith('save failed')
  })
})
