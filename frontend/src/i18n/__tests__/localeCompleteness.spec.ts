import { readFileSync, readdirSync, statSync } from 'node:fs'
import { extname, join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

type LocaleTree = Record<string, unknown>

type StaticKeyReference = {
  file: string
  key: string
}

const sourceRoot = join(process.cwd(), 'src')
const ignoredDirectories = new Set(['__tests__', 'i18n'])
const supportedExtensions = new Set(['.ts', '.vue'])
const staticTranslationCall = /(?:\$t|\bt|\btranslate|i18n\.global\.t)\(\s*(['"`])([A-Za-z][A-Za-z0-9_.-]*)\1/g

function collectSourceFiles(directory: string): string[] {
  return readdirSync(directory)
    .flatMap((entry) => {
      const path = join(directory, entry)
      if (statSync(path).isDirectory()) {
        return ignoredDirectories.has(entry) ? [] : collectSourceFiles(path)
      }
      return supportedExtensions.has(extname(entry)) ? [path] : []
    })
    .sort()
}

function collectStaticTranslationKeys(): StaticKeyReference[] {
  const references: StaticKeyReference[] = []
  for (const file of collectSourceFiles(sourceRoot)) {
    const source = readFileSync(file, 'utf8')
    for (const match of source.matchAll(staticTranslationCall)) {
      references.push({
        file: relative(sourceRoot, file).replaceAll('\\', '/'),
        key: match[2]!
      })
    }
  }
  return references
}

function resolveLocaleKey(locale: LocaleTree, key: string): unknown {
  return key.split('.').reduce<unknown>((value, segment) => {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined
    return (value as LocaleTree)[segment]
  }, locale)
}

function collectLeafKeys(value: unknown, prefix = ''): string[] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return prefix ? [prefix] : []
  }
  return Object.entries(value as LocaleTree).flatMap(([key, child]) =>
    collectLeafKeys(child, prefix ? `${prefix}.${key}` : key)
  )
}

function collectPlaceholders(message: string): string[] {
  return [...new Set(
    [...message.matchAll(/\{([A-Za-z_][A-Za-z0-9_]*)\}/g)].map((match) => match[1]!)
  )].sort()
}

const references = collectStaticTranslationKeys()
const uniqueKeys = [...new Set(references.map(({ key }) => key))]
  .filter((key) => !key.endsWith('.'))
  .sort()
const dynamicPrefixes = [...new Set(references.map(({ key }) => key).filter((key) => key.endsWith('.')))]
  .sort()

describe('locale completeness', () => {
  it.each([
    ['zh', zh],
    ['en', en]
  ] as const)('%s resolves every statically referenced production key', (localeName, locale) => {
    const missing = uniqueKeys
      .filter((key) => {
        const value = resolveLocaleKey(locale, key)
        return typeof value !== 'string' || value.trim() === '' || value === key
      })
      .map((key) => ({
        key,
        files: references.filter((reference) => reference.key === key).map(({ file }) => file)
      }))

    expect(missing, `${localeName} has missing or empty static translation keys`).toEqual([])
  })

  it.each([
    ['zh', zh],
    ['en', en]
  ] as const)('%s resolves every statically discoverable dynamic-key prefix', (_localeName, locale) => {
    const missingPrefixes = dynamicPrefixes.filter((prefix) => {
      const namespace = resolveLocaleKey(locale, prefix.slice(0, -1))
      return !namespace || typeof namespace !== 'object' || Array.isArray(namespace)
    })

    expect(missingPrefixes).toEqual([])
  })

  it('keeps Chinese and English locale leaf keys symmetric', () => {
    const zhKeys = collectLeafKeys(zh).sort()
    const enKeys = collectLeafKeys(en).sort()
    const onlyZh = zhKeys.filter((key) => !enKeys.includes(key))
    const onlyEn = enKeys.filter((key) => !zhKeys.includes(key))

    expect({ onlyZh, onlyEn }).toEqual({ onlyZh: [], onlyEn: [] })
  })

  it('keeps Chinese and English interpolation placeholders symmetric', () => {
    const mismatches = collectLeafKeys(zh)
      .map((key) => {
        const zhMessage = resolveLocaleKey(zh, key)
        const enMessage = resolveLocaleKey(en, key)
        if (typeof zhMessage !== 'string' || typeof enMessage !== 'string') return undefined
        const zhPlaceholders = collectPlaceholders(zhMessage)
        const enPlaceholders = collectPlaceholders(enMessage)
        if (JSON.stringify(zhPlaceholders) === JSON.stringify(enPlaceholders)) return undefined
        return { key, zhPlaceholders, enPlaceholders }
      })
      .filter(Boolean)

    expect(mismatches).toEqual([])
  })
})
