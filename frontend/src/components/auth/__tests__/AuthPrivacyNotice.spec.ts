import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import AuthPrivacyNotice from '@/components/auth/AuthPrivacyNotice.vue'

const messages: Record<string, string> = {
  'auth.privacyCommitment': '隐私承诺',
  'auth.privacyDescription': '我们不记录任何 API 请求内容，不向第三方出售用户数据，也不会将用户数据用于广告投放或模型训练。系统仅在提供服务所必需的范围内，对相关数据进行临时处理。',
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => messages[key] ?? key,
  }),
}))

describe('AuthPrivacyNotice', () => {
  it('presents the privacy commitment as an accessible note', () => {
    const wrapper = mount(AuthPrivacyNotice, {
      global: {
        stubs: {
          Icon: { template: '<span />' },
        },
      },
    })
    const notice = wrapper.get('[data-test="auth-privacy-notice"]')

    expect(notice.attributes('role')).toBe('note')
    expect(notice.attributes('aria-labelledby')).toBe('auth-privacy-title')
    expect(notice.text()).toContain('隐私承诺')
    expect(notice.text()).toContain('不记录任何 API 请求内容')
    expect(notice.text()).toContain('不向第三方出售用户数据')
    expect(notice.text()).toContain('不会将用户数据用于广告投放或模型训练')
  })
})
