import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountTableFilters from '../AccountTableFilters.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: `
    <div data-test="select">
      <button
        v-for="option in options"
        :key="String(option.value)"
        type="button"
        :data-test="'option-' + String(option.value)"
        @click="$emit('update:modelValue', option.value); $emit('change', option.value, option)"
      >
        {{ option.label }}
      </button>
    </div>
  `
}

const SearchInputStub = {
  props: ['modelValue', 'placeholder'],
  emits: ['update:modelValue', 'search'],
  template: '<input data-test="search" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
}

const mountFilters = (filters: Record<string, unknown>) => mount(AccountTableFilters, {
  props: {
    searchQuery: '',
    filters,
    groups: []
  },
  global: {
    stubs: {
      Select: SelectStub,
      SearchInput: SearchInputStub
    }
  }
})

describe('AccountTableFilters', () => {
  it('shows the OpenAI plan filter only for OpenAI accounts', () => {
    const wrapper = mountFilters({ platform: 'openai', openai_plan_type: '' })

    expect(wrapper.text()).toContain('admin.accounts.openAIPlanTypes.all')
    expect(wrapper.text()).toContain('Plus')
    expect(wrapper.text()).toContain('Team')
    expect(wrapper.text()).toContain('Free')
  })

  it('clears the OpenAI plan filter when switching away from OpenAI', async () => {
    const wrapper = mountFilters({ platform: 'openai', openai_plan_type: 'plus' })

    await wrapper.get('[data-test="option-anthropic"]').trigger('click')

    const emitted = wrapper.emitted('update:filters')
    expect(emitted).toBeTruthy()
    expect(emitted?.[0]?.[0]).toMatchObject({
      platform: 'anthropic',
      openai_plan_type: ''
    })
  })
})
