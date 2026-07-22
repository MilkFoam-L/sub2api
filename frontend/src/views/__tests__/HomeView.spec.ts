import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

const fetchPublicSettings = vi.fn()
const checkAuth = vi.fn()

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh-CN' }
    })
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    isAuthenticated: false,
    isAdmin: false,
    user: null,
    checkAuth
  }),
  useAppStore: () => ({
    cachedPublicSettings: null,
    siteName: 'Sub2API',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings
  })
}))

const mountHome = () => mount(HomeView, {
  global: {
    stubs: {
      RouterLink: {
        props: ['to'],
        template: '<a :href="typeof to === \'string\' ? to : \'/\'"><slot /></a>'
      },
      LocaleSwitcher: { template: '<div />' },
      Icon: { template: '<span />' }
    }
  }
})

describe('HomeView hero layout', () => {
  beforeEach(() => {
    localStorage.clear()
    fetchPublicSettings.mockReset()
    checkAuth.mockReset()
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn().mockReturnValue({ matches: false })
    })
  })

  it('shrinks the navigation on scroll and expands again at the top', async () => {
    const wrapper = mountHome()

    expect(wrapper.classes()).toContain('overflow-x-clip')
    expect(wrapper.classes()).not.toContain('overflow-x-hidden')
    expect(wrapper.classes()).not.toContain('overflow-hidden')
    expect(wrapper.get('[data-test="home-nav"]').classes()).toContain('sticky')
    expect(wrapper.get('[data-test="home-nav"]').attributes('data-state')).toBe('expanded')

    Object.defineProperty(window, 'scrollY', { configurable: true, value: 96 })
    window.dispatchEvent(new Event('scroll'))
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-test="home-nav"]').attributes('data-state')).toBe('condensed')

    Object.defineProperty(window, 'scrollY', { configurable: true, value: 0 })
    window.dispatchEvent(new Event('scroll'))
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-test="home-nav"]').attributes('data-state')).toBe('expanded')
  })

  it('uses Telegram community links instead of GitHub buttons', () => {
    const wrapper = mountHome()
    const links = wrapper.findAll('a[href="https://t.me/+6k_-l_zbIHpkZGNh"]')

    expect(links.length).toBeGreaterThan(0)
    expect(links.some((link) => link.classes().includes('btn-primary'))).toBe(true)
    expect(wrapper.text()).toContain('交流群')
    expect(wrapper.text()).not.toContain('GitHub')
  })

  it('keeps endpoint feature card text inside its borders', () => {
    const wrapper = mountHome()
    const card = wrapper.get('[data-test="endpoint-feature-card-openai"]')
    const description = card.get('[data-test="endpoint-feature-description"]')

    expect(card.classes()).toContain('overflow-hidden')
    expect(card.classes()).toContain('min-w-0')
    expect(description.classes()).toContain('break-words')
  })

  it('places latency in the left hero stats instead of the right endpoint card', () => {
    const wrapper = mountHome()

    expect(wrapper.get('[data-test="hero-stats"]').text()).toContain('68ms')
    expect(wrapper.get('[data-test="hero-stats"]').text()).toContain('平均路由延迟')
    expect(wrapper.get('[data-test="endpoint-card"]').text()).not.toContain('68ms')
  })

  it('switches Base URL between OpenAI /v1 and Claude root endpoint', async () => {
    const wrapper = mountHome()
    const origin = window.location.origin

    expect(wrapper.get('[data-test="endpoint-url"]').text()).toBe(`${origin}/v1`)

    await wrapper.get('[data-test="endpoint-tab-claude"]').trigger('click')

    expect(wrapper.get('[data-test="endpoint-url"]').text()).toBe(origin)
    expect(wrapper.text()).toContain('Claude')
    expect(wrapper.text()).toContain('无 /v1 后缀')

    await wrapper.get('[data-test="endpoint-tab-openai"]').trigger('click')

    expect(wrapper.get('[data-test="endpoint-url"]').text()).toBe(`${origin}/v1`)
  })
})
