import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserTokenLeaderboardView from '../UserTokenLeaderboardView.vue'

const { getUserBreakdown, routerPush } = vi.hoisted(() => ({
  getUserBreakdown: vi.fn(),
  routerPush: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getUserBreakdown
    }
  }
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const formatLocalDate = (date: Date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function mountView() {
  return mount(UserTokenLeaderboardView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        DateRangePicker: true,
        Select: true,
        UsageFilters: true,
        UserTokenLeaderboard: true,
      }
    }
  })
}

describe('UserTokenLeaderboardView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-06-16T15:30:00'))
    getUserBreakdown.mockReset()
    routerPush.mockReset()
    getUserBreakdown.mockResolvedValue({ users: [] })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('默认按今天查询 token 排行榜', async () => {
    mountView()
    await flushPromises()

    const today = formatLocalDate(new Date('2026-06-16T15:30:00'))
    expect(getUserBreakdown).toHaveBeenCalledWith(expect.objectContaining({
      start_date: today,
      end_date: today,
      sort_by: 'tokens',
      limit: 10,
    }))
  })
})
