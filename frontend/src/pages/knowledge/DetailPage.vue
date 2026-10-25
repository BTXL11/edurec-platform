<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getKnowledgePoint } from '@/api/knowledge'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import KnowledgeGraph from '@/components/knowledge/KnowledgeGraph.vue'
import type { KnowledgeDetail, KnowledgeNode } from '@/types'

const route = useRoute()
const router = useRouter()
const code = computed(() => String(route.params.code ?? ''))

const detail = ref<KnowledgeDetail | null>(null)
const loading = ref(true)
const error = ref('')

const VISIBLE_CONCEPTS = 30
const visibleConcepts = computed(() => (detail.value?.concepts ?? []).slice(0, VISIBLE_CONCEPTS))
const hiddenConceptCount = computed(() =>
  Math.max(0, (detail.value?.concepts.length ?? 0) - visibleConcepts.value.length),
)

async function load() {
  if (!code.value) return
  loading.value = true
  error.value = ''
  try {
    detail.value = await getKnowledgePoint(code.value, 12)
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
    detail.value = null
  } finally {
    loading.value = false
  }
}

function openNode(node: KnowledgeNode) {
  if (node.code === code.value) return
  router.push({ name: 'knowledge-detail', params: { code: node.code } })
}

onMounted(load)
watch(code, load)
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <RouterLink :to="{ name: 'knowledge' }" class="text-sm text-primary hover:underline">
      ← 返回知识点导航
    </RouterLink>

    <div v-if="loading" class="mt-8">
      <el-skeleton :rows="8" animated />
    </div>
    <el-empty v-else-if="error" :description="error" class="mt-16" />

    <template v-else-if="detail">
      <header class="mt-4">
        <div class="flex flex-wrap items-center gap-3">
          <h1 class="text-2xl font-bold text-ink">{{ detail.point.name }}</h1>
          <el-tag size="small" type="primary">第 {{ detail.point.level }} 层</el-tag>
          <span class="text-xs text-ink-muted">
            课程资源 {{ detail.point.resource_count }} · 概念 {{ detail.point.concept_count }}
          </span>
        </div>
        <p class="mt-2 max-w-3xl text-sm leading-relaxed text-ink-secondary">
          {{ detail.point.description || '暂无描述' }}
        </p>
      </header>

      <section class="mt-8 rounded-lg border border-border bg-surface p-5">
        <div class="flex items-center justify-between">
          <h2 class="text-base font-semibold text-ink">先修关系</h2>
          <span class="text-xs text-ink-muted">箭头表示「先学 → 后学」，点击节点可切换</span>
        </div>
        <KnowledgeGraph
          class="mt-4"
          :nodes="detail.nodes"
          :edges="detail.edges"
          :current-id="detail.point.id"
          @select="openNode"
        />
      </section>

      <section class="mt-6 rounded-lg border border-border bg-surface p-5">
        <div class="flex items-center justify-between">
          <h2 class="text-base font-semibold text-ink">概念图谱</h2>
          <span class="text-xs text-ink-muted">
            来自 EMBEDD-ER / MOOCCube{{ detail.concepts.length ? ` · 共 ${detail.concepts.length} 个概念` : '' }}
          </span>
        </div>

        <template v-if="visibleConcepts.length > 0">
          <div class="mt-4 flex flex-wrap gap-2">
            <el-tooltip
              v-for="concept in visibleConcepts"
              :key="concept.id"
              :content="concept.explanation || `${concept.name}（${concept.field}）`"
              placement="top"
            >
              <el-tag type="info" effect="plain">
                {{ concept.name }}
                <span class="ml-1 text-[11px] text-ink-muted">L{{ concept.level }}</span>
              </el-tag>
            </el-tooltip>
          </div>
          <p v-if="hiddenConceptCount > 0" class="mt-3 text-xs text-ink-muted">
            另有 {{ hiddenConceptCount }} 个概念未展示
          </p>
          <p class="mt-3 text-xs text-ink-muted">
            概念先修边 {{ detail.concept_edges.length }} 条；同一层级的概念可以并行学习。
          </p>
        </template>
        <el-empty
          v-else
          description="该学科暂未匹配到概念层（MOOCCube 未覆盖），先按学科先修顺序学习即可"
          :image-size="80"
          class="mt-2"
        />
      </section>

      <section class="mt-6">
        <div class="flex items-end justify-between">
          <h2 class="text-lg font-semibold text-ink">课程资源</h2>
          <span class="text-xs text-ink-muted">共 {{ detail.resource_total }} 条</span>
        </div>

        <div
          v-if="detail.resources.length > 0"
          class="mt-4 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"
        >
          <ResourceCard
            v-for="resource in detail.resources"
            :key="resource.id"
            :resource="resource"
          />
        </div>
        <el-empty
          v-else
          description="该知识点下暂无资源，可先到搜索页按关键词采集"
          class="mt-4"
        />
      </section>
    </template>
  </div>
</template>
