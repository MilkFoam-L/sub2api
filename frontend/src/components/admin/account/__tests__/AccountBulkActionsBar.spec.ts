import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountBulkActionsBar', () => {
  it('shows only filtered-result actions when no accounts are selected', () => {
    const wrapper = mount(AccountBulkActionsBar, { props: { selectedIds: [] } })

    expect(wrapper.find('[data-test="selected-delete"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="filtered-delete"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="filtered-edit"]').exists()).toBe(true)
  })

  it('shows only selected-account actions when accounts are selected', async () => {
    const wrapper = mount(AccountBulkActionsBar, { props: { selectedIds: [7, 11] } })

    expect(wrapper.get('[data-test="selected-delete"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="filtered-delete"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="filtered-edit"]').exists()).toBe(false)

    await wrapper.get('[data-test="selected-delete"]').trigger('click')
    expect(wrapper.emitted('delete')).toHaveLength(1)
  })

  it('uses wrapping responsive layout for narrow screens', () => {
    const wrapper = mount(AccountBulkActionsBar, { props: { selectedIds: [7] } })

    expect(wrapper.classes()).toEqual(expect.arrayContaining(['flex-col', 'sm:flex-row']))
    expect(wrapper.get('[data-test="selected-actions"]').classes()).toContain('flex-wrap')
  })
})
