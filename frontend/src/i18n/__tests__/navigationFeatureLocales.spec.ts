import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const requiredKeys = [
  'nav.tokenLeaderboard',
  'nav.modelMarket',
  'modelMarket.title',
  'modelMarket.description',
  'modelMarket.searchPlaceholder',
  'modelMarket.filters.allPlatforms',
  'modelMarket.filters.allChannels',
  'modelMarket.filters.allPricing',
  'modelMarket.filters.withPricing',
  'modelMarket.filters.withoutPricing',
  'modelMarket.summary.groups',
  'modelMarket.summary.models',
  'modelMarket.summary.channels',
  'modelMarket.columns.group',
  'modelMarket.columns.platform',
  'modelMarket.columns.channels',
  'modelMarket.columns.models',
  'modelMarket.columns.pricing',
  'modelMarket.empty',
  'modelMarket.pricingVaries',
  'modelMarket.pricingConfigured',
  'admin.usage.tokenLeaderboardTitle',
  'admin.usage.tokenLeaderboardDescription',
  'admin.usage.tokenLeaderboardLimit',
  'admin.usage.tokenLeaderboardTop',
  'admin.usage.tokenLeaderboardLoadFailed',
  'admin.usage.tokenLeaderboardNoData',
  'admin.usage.tokenLeaderboardRank',
  'admin.usage.tokenLeaderboardUser',
  'admin.usage.tokenLeaderboardTokens',
  'admin.usage.tokenLeaderboardRequests',
  'admin.usage.tokenLeaderboardCost',
] as const

function resolveLocaleKey(locale: unknown, key: string): unknown {
  return key.split('.').reduce<unknown>((value, segment) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[segment]
  }, locale)
}

describe.each([
  ['zh', zh],
  ['en', en],
])('%s navigation feature locale text', (_localeName, locale) => {
  it.each(requiredKeys)('resolves %s to visible text', (key) => {
    const value = resolveLocaleKey(locale, key)

    expect(value, `${key} should exist`).toEqual(expect.any(String))
    expect((value as string).trim()).not.toBe('')
    expect(value).not.toBe(key)
  })
})
