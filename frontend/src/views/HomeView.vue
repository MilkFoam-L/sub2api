<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe v-if="isHomeContentUrl" :src="homeContent.trim()" class="h-screen w-full border-0" allowfullscreen />
    <div v-else v-html="homeContent"></div>
  </div>

  <div v-else class="relative min-h-screen overflow-x-hidden bg-background text-foreground">
    <div class="pointer-events-none absolute inset-0">
      <div class="absolute inset-0 bg-[radial-gradient(circle_at_15%_10%,hsl(var(--primary)/0.16),transparent_28%),radial-gradient(circle_at_85%_20%,hsl(var(--accent)/0.32),transparent_30%),linear-gradient(hsl(var(--border)/0.35)_1px,transparent_1px),linear-gradient(90deg,hsl(var(--border)/0.35)_1px,transparent_1px)] bg-[size:auto,auto,72px_72px,72px_72px]"></div>
    </div>

    <header
      data-test="home-nav"
      :data-state="navCondensed ? 'condensed' : 'expanded'"
      class="sticky top-0 z-30 px-4 transition-all duration-500 ease-out"
      :class="navCondensed ? 'py-2' : 'py-4'"
    >
      <nav
        class="mx-auto flex items-center justify-between gap-4 border transition-all duration-500 ease-out"
        :class="navCondensed
          ? 'max-w-5xl rounded-full border-border/80 bg-background/85 px-4 py-2 shadow-2xl shadow-primary/10 backdrop-blur-2xl'
          : 'max-w-7xl rounded-[1.75rem] border-transparent bg-transparent px-0 py-0 shadow-none'"
      >
        <router-link to="/" class="flex items-center gap-3">
          <span
            class="flex overflow-hidden border border-border bg-card shadow-sm transition-all duration-500"
            :class="navCondensed ? 'h-8 w-8 rounded-xl' : 'h-10 w-10 rounded-2xl'"
          >
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <span class="hidden text-sm font-semibold sm:block">{{ siteName }}</span>
        </router-link>
        <div class="hidden items-center gap-6 text-sm text-muted-foreground md:flex">
          <a href="#models" class="hover:text-foreground">{{ text.nav.models }}</a>
          <a href="#features" class="hover:text-foreground">{{ text.nav.features }}</a>
          <a href="#workflow" class="hover:text-foreground">{{ text.nav.workflow }}</a>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener" class="hover:text-foreground">{{ t('home.docs') }}</a>
        </div>
        <div class="flex items-center gap-2">
          <LocaleSwitcher />
          <button class="rounded-xl p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground" @click="toggleTheme">
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="btn btn-primary btn-sm">
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative z-10">
      <section class="mx-auto grid max-w-7xl gap-12 px-5 pb-16 pt-14 lg:grid-cols-[1.06fr_0.94fr] lg:pb-24 lg:pt-20">
        <div class="flex flex-col justify-center">
          <div class="mb-5 inline-flex w-fit items-center gap-2 rounded-full border border-border bg-card/80 px-3 py-1 text-xs font-medium text-muted-foreground shadow-sm">
            <span class="h-2 w-2 rounded-full bg-primary"></span>{{ text.hero.badge }}
          </div>
          <h1 class="max-w-4xl text-4xl font-semibold tracking-tight text-foreground sm:text-5xl lg:text-6xl">
            {{ text.hero.title }}
          </h1>
          <p class="mt-6 max-w-2xl text-base leading-8 text-muted-foreground sm:text-lg">
            {{ text.hero.desc }}
          </p>
          <div class="mt-8 flex flex-col gap-3 sm:flex-row">
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="btn btn-primary px-6 py-3 text-sm">
              {{ isAuthenticated ? t('home.goToDashboard') : text.hero.primary }}
              <Icon name="arrowRight" size="sm" class="ml-2" />
            </router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener" class="btn btn-secondary px-6 py-3 text-sm">{{ text.hero.secondary }}</a>
            <a
              :href="communityUrl"
              target="_blank"
              rel="noopener"
              class="btn btn-primary group relative overflow-hidden px-6 py-3 text-sm shadow-2xl shadow-primary/30 ring-2 ring-primary/40 transition hover:-translate-y-0.5 hover:shadow-primary/40"
            >
              <span class="absolute inset-0 bg-gradient-to-r from-primary via-primary to-accent opacity-95"></span>
              <span class="relative flex items-center gap-2 text-primary-foreground">
                <span class="h-2 w-2 rounded-full bg-primary-foreground shadow-[0_0_14px_hsl(var(--primary-foreground))]"></span>
                {{ text.hero.community }}
              </span>
            </a>
          </div>
          <div data-test="hero-stats" class="mt-10 grid max-w-2xl grid-cols-2 gap-3 sm:grid-cols-4">
            <div v-for="item in text.stats" :key="item.label" class="rounded-2xl border border-border bg-card/70 p-4 shadow-sm">
              <div class="text-2xl font-semibold">{{ item.value }}</div>
              <div class="mt-1 text-xs text-muted-foreground">{{ item.label }}</div>
            </div>
          </div>
        </div>

        <div data-test="endpoint-card" class="relative min-w-0">
          <div class="absolute -inset-1 rounded-[2.25rem] bg-[linear-gradient(135deg,hsl(var(--primary)/0.55),hsl(var(--accent)/0.28),hsl(var(--border)/0.95))] opacity-80 blur-sm"></div>
          <div class="relative min-w-0 overflow-hidden rounded-[2rem] border border-primary/30 bg-card/95 p-[1px] shadow-[0_28px_90px_hsl(var(--primary)/0.22)] backdrop-blur">
            <div class="min-w-0 overflow-hidden rounded-[1.95rem] border border-border/80 bg-background/95 p-4">
              <div class="min-w-0 overflow-hidden rounded-[1.5rem] border border-primary/20 bg-card/70 p-5 shadow-inner shadow-primary/5">
                <div class="mb-4 flex min-w-0 items-center justify-between gap-3">
                  <div class="min-w-0">
                    <p class="text-xs uppercase tracking-[0.3em] text-muted-foreground">Base URL</p>
                    <h2 class="mt-1 break-words text-lg font-semibold">{{ text.endpoint.title }}</h2>
                  </div>
                  <span class="rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">Online</span>
                </div>
                <div class="mb-4 grid min-w-0 grid-cols-2 gap-2 overflow-hidden rounded-2xl border border-border bg-muted/40 p-1">
                  <button
                    v-for="option in endpointProviderTabs"
                    :key="option.id"
                    type="button"
                    :data-test="`endpoint-tab-${option.id}`"
                    class="min-w-0 rounded-xl px-3 py-2 text-left transition"
                    :class="endpointProvider === option.id ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
                    @click="selectEndpointProvider(option.id)"
                  >
                    <span class="block text-sm font-semibold">{{ option.label }}</span>
                    <span class="mt-0.5 block text-[11px]">{{ option.hint }}</span>
                  </button>
                </div>
                <div class="flex items-center gap-2 rounded-2xl border border-border bg-muted/40 p-3">
                  <code data-test="endpoint-url" class="min-w-0 flex-1 truncate text-sm text-foreground">{{ activeEndpointUrl }}</code>
                  <button class="rounded-xl bg-card p-2 text-muted-foreground hover:text-foreground" @click="copyEndpoint"><Icon name="copy" size="sm" /></button>
                </div>
                <p class="mt-2 break-words text-xs leading-5 text-muted-foreground">{{ activeEndpointDescription }}</p>
                <div class="mt-5 grid min-w-0 gap-3 sm:grid-cols-2">
                  <div
                    v-for="card in text.heroCards"
                    :key="card.title"
                    :data-test="card.id === 'openai-compatible' ? 'endpoint-feature-card-openai' : undefined"
                    class="min-w-0 overflow-hidden rounded-2xl border border-border bg-card p-4"
                  >
                    <p class="text-sm font-semibold break-words">{{ card.title }}</p>
                    <p data-test="endpoint-feature-description" class="mt-2 break-words text-xs leading-5 text-muted-foreground [overflow-wrap:anywhere]">{{ card.desc }}</p>
                  </div>
                </div>
                <div class="mt-5 min-w-0 overflow-hidden rounded-2xl border border-border bg-card p-4 font-mono text-xs leading-6 text-muted-foreground">
                  <p v-for="endpoint in activeEndpointExamples" :key="endpoint.path" class="break-all">
                    <span class="text-primary">{{ endpoint.method }}</span> {{ endpoint.path }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section id="models" class="border-y border-border bg-card/40 px-5 py-10">
        <div class="mx-auto max-w-7xl">
          <div class="mb-6 flex flex-col justify-between gap-3 sm:flex-row sm:items-end">
            <div><p class="text-sm font-medium text-primary">{{ text.models.kicker }}</p><h2 class="mt-2 text-3xl font-semibold">{{ text.models.title }}</h2></div>
            <p class="max-w-xl text-sm leading-6 text-muted-foreground">{{ text.models.desc }}</p>
          </div>
          <div class="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-8">
            <div v-for="model in modelChips" :key="model" class="rounded-2xl border border-border bg-background px-4 py-4 text-center text-sm font-medium shadow-sm">{{ model }}</div>
          </div>
        </div>
      </section>

      <section id="features" class="mx-auto max-w-7xl px-5 py-16 lg:py-24">
        <div class="mb-10 max-w-2xl"><p class="text-sm font-medium text-primary">{{ text.features.kicker }}</p><h2 class="mt-2 text-3xl font-semibold">{{ text.features.title }}</h2></div>
        <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <article v-for="feature in text.features.items" :key="feature.title" class="rounded-3xl border border-border bg-card p-6 shadow-sm transition hover:-translate-y-1 hover:shadow-lg">
            <div class="mb-5 flex h-11 w-11 items-center justify-center rounded-2xl bg-primary/10 text-primary"><Icon :name="feature.icon" size="md" /></div>
            <h3 class="text-lg font-semibold">{{ feature.title }}</h3>
            <p class="mt-3 text-sm leading-6 text-muted-foreground">{{ feature.desc }}</p>
          </article>
        </div>
      </section>

      <section id="workflow" class="mx-auto max-w-7xl px-5 pb-16">
        <div class="rounded-[2rem] border border-border bg-card p-6 shadow-sm lg:p-10">
          <div class="grid gap-8 lg:grid-cols-[0.8fr_1.2fr]">
            <div><p class="text-sm font-medium text-primary">{{ text.workflow.kicker }}</p><h2 class="mt-2 text-3xl font-semibold">{{ text.workflow.title }}</h2><p class="mt-4 text-sm leading-6 text-muted-foreground">{{ text.workflow.desc }}</p></div>
            <div class="grid gap-4 md:grid-cols-3">
              <div v-for="(step, index) in text.workflow.steps" :key="step.title" class="rounded-2xl border border-border bg-background p-5">
                <span class="text-xs font-semibold text-primary">0{{ index + 1 }}</span>
                <h3 class="mt-4 font-semibold">{{ step.title }}</h3>
                <p class="mt-2 text-sm leading-6 text-muted-foreground">{{ step.desc }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="mx-auto max-w-7xl px-5 pb-20">
        <div class="rounded-[2rem] border border-primary/20 bg-primary px-6 py-10 text-center text-primary-foreground shadow-2xl shadow-primary/20">
          <h2 class="text-3xl font-semibold">{{ text.cta.title }}</h2>
          <p class="mx-auto mt-3 max-w-2xl text-sm leading-6 text-primary-foreground/80">{{ text.cta.desc }}</p>
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="mt-6 inline-flex rounded-2xl bg-primary-foreground px-6 py-3 text-sm font-semibold text-primary hover:bg-primary-foreground/90">{{ text.cta.button }}</router-link>
        </div>
      </section>
    </main>

    <footer class="relative z-10 border-t border-border px-5 py-8">
      <div class="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 text-sm text-muted-foreground sm:flex-row">
        <p>© {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        <div class="flex gap-4"><a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener" class="hover:text-foreground">{{ t('home.docs') }}</a>          <a :href="communityUrl" target="_blank" rel="noopener" class="hover:text-foreground">{{ text.hero.community }}</a>
</div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || '')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})
const isDark = ref(document.documentElement.classList.contains('dark'))
const navCondensed = ref(false)
const communityUrl = 'https://t.me/+6k_-l_zbIHpkZGNh'
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const currentYear = computed(() => new Date().getFullYear())
type EndpointProvider = 'openai' | 'claude'
const endpointProvider = ref<EndpointProvider>('openai')
const siteOrigin = computed(() => typeof window === 'undefined' ? '' : window.location.origin)
const openAIBaseUrl = computed(() => siteOrigin.value ? `${siteOrigin.value}/v1` : '/v1')
const claudeBaseUrl = computed(() => siteOrigin.value || '/')
const activeEndpointUrl = computed(() => endpointProvider.value === 'openai' ? openAIBaseUrl.value : claudeBaseUrl.value)

const zh = computed(() => String(locale.value).toLowerCase().startsWith('zh'))
const pick = (cn: string, en: string) => zh.value ? cn : en
const endpointProviderTabs = computed(() => [
  { id: 'openai' as const, label: 'OpenAI', hint: pick('Base URL 带 /v1', 'Base URL with /v1') },
  { id: 'claude' as const, label: 'Claude', hint: pick('无 /v1 后缀', 'No /v1 suffix') }
])
const activeEndpointDescription = computed(() => endpointProvider.value === 'openai'
  ? pick('OpenAI 兼容接口需要使用带 /v1 的 Base URL。', 'OpenAI-compatible clients should use the /v1 Base URL.')
  : pick('Claude Messages 接口使用站点根地址作为 Base URL，无 /v1 后缀。', 'Claude Messages clients use the root Base URL without the /v1 suffix.')
)
const activeEndpointExamples = computed(() => endpointProvider.value === 'openai'
  ? [
      { method: 'POST', path: '/chat/completions' },
      { method: 'GET', path: '/models' }
    ]
  : [
      { method: 'POST', path: '/v1/messages' },
      { method: 'GET', path: '/v1/models' }
    ]
)
const modelChips = ['OpenAI', 'Claude', 'Gemini', 'DeepSeek', 'Qwen', 'Azure AI', 'Moonshot', 'Minimax']
type HomeFeatureIcon = 'server' | 'swap' | 'chart' | 'shield'
const featureIcon = (icon: HomeFeatureIcon) => icon
const text = computed(() => ({
  nav: { models: pick('模型生态', 'Models'), features: pick('核心能力', 'Features'), workflow: pick('接入流程', 'Workflow') },
  hero: {
    badge: pick('企业级 AI API 网关 · 多账号池 · 统一计费', 'Enterprise AI API gateway · account pool · unified billing'),
    title: pick('一站接入全球 AI 能力，像调用 OpenAI 一样调用所有模型', 'Connect global AI models through one OpenAI-compatible gateway'),
    desc: siteSubtitle.value || pick('Sub2API 将 OpenAI、Claude、Gemini、DeepSeek、Qwen 等平台聚合为统一接口，自动调度账号池、监控用量、控制成本，让团队专注构建产品。', 'Sub2API unifies OpenAI, Claude, Gemini, DeepSeek, Qwen and more behind one endpoint with account scheduling, usage visibility, and cost controls.'),
    primary: pick('获取 API Key', 'Get API Key'),
    secondary: pick('查看接入文档', 'Read Docs'),
    community: pick('交流群', 'Community')
  },
  stats: [{ value: '30+', label: pick('模型与端点', 'models') }, { value: '99.9%', label: pick('可用性目标', 'uptime') }, { value: '68ms', label: pick('平均路由延迟', 'avg latency') }, { value: '1 URL', label: pick('统一入口', 'base URL') }],
  endpoint: { title: pick('复制 Base URL，立即切换', 'Copy Base URL and switch fast') },
  heroCards: [
    { id: 'smart-routing', title: pick('智能账号调度', 'Smart routing'), desc: pick('按状态、倍率、并发和分组自动分流。', 'Dispatch by status, rate, concurrency, and group.') },
    { id: 'full-visibility', title: pick('全链路可观测', 'Full visibility'), desc: pick('请求、费用、模型、分组和账号状态集中查看。', 'Track requests, costs, models, groups, and accounts.') },
    { id: 'safe-rate-limits', title: pick('安全限流', 'Safe rate limits'), desc: pick('API Key、分组、模型白名单和预算策略统一管理。', 'Manage keys, groups, model allowlists, and budgets.') },
    { id: 'openai-compatible', title: pick('OpenAI 兼容', 'OpenAI compatible'), desc: pick('兼容 /v1/chat/completions、/v1/messages 与 /v1/models。', 'Supports chat completions, messages, and models APIs.') }
  ],
  models: { kicker: pick('模型生态', 'Model ecosystem'), title: pick('一个入口，覆盖主流大模型平台', 'One entry point for mainstream AI providers'), desc: pick('保留熟悉的 OpenAI 调用方式，同时接入多个上游平台与账号池。', 'Keep OpenAI-style calls while routing to multiple upstream platforms and account pools.') },
  features: { kicker: pick('为什么选择 Sub2API', 'Why Sub2API'), title: pick('从接入、调度到运营的一体化网关', 'A complete gateway from integration to operations'), items: [
    { icon: featureIcon('server'), title: pick('统一 API 入口', 'Unified API'), desc: pick('一套 Base URL 和 Key 连接多平台模型，降低迁移成本。', 'Use one Base URL and key across providers with lower migration cost.') },
    { icon: featureIcon('swap'), title: pick('账号池调度', 'Account pool'), desc: pick('自动规避异常账号，按负载、分组和优先级调度。', 'Avoid unhealthy accounts and route by load, group, and priority.') },
    { icon: featureIcon('chart'), title: pick('实时用量计费', 'Usage billing'), desc: pick('按用户、Key、分组、模型追踪用量与成本。', 'Track usage and cost by user, key, group, and model.') },
    { icon: featureIcon('shield'), title: pick('权限与风控', 'Access control'), desc: pick('支持模型白名单、额度、限速和审计，适合团队协作。', 'Model allowlists, quotas, rate limits, and audit logs for teams.') }
  ] },
  workflow: { kicker: pick('快速上线', 'Launch fast'), title: pick('三步完成从订阅到 API 服务化', 'Turn subscriptions into API service in three steps'), desc: pick('从账号接入到稳定运营，所有关键动作都在控制台完成。', 'Operate account onboarding, routing, and monitoring from the console.'), steps: [
    { title: pick('导入账号与代理', 'Import accounts'), desc: pick('批量接入 OpenAI、Claude、Gemini 等账号与代理。', 'Bulk import accounts and proxies for providers.') },
    { title: pick('配置分组和策略', 'Configure policies'), desc: pick('设置分组倍率、模型白名单、并发和优先级。', 'Set group pricing, allowlists, concurrency, and priority.') },
    { title: pick('交付统一接口', 'Ship one endpoint'), desc: pick('将 Base URL 与 Key 交付给应用或团队成员。', 'Share one Base URL and key with apps or teammates.') }
  ] },
  cta: { title: pick('准备把 AI 能力交付给团队了吗？', 'Ready to deliver AI capabilities to your team?'), desc: pick('用统一网关管理模型、账号、成本与安全策略，今天就开始。', 'Manage models, accounts, costs, and policies through one gateway.'), button: pick('进入控制台', 'Open Console') }
}))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}
function selectEndpointProvider(provider: EndpointProvider) {
  endpointProvider.value = provider
}
async function copyEndpoint() {
  try { await navigator.clipboard?.writeText(activeEndpointUrl.value) } catch { /* best effort */ }
}
function updateNavState() {
  navCondensed.value = window.scrollY > 32
}
onMounted(() => {
  initTheme()
  updateNavState()
  window.addEventListener('scroll', updateNavState, { passive: true })
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()
})
onUnmounted(() => {
  window.removeEventListener('scroll', updateNavState)
})
</script>
