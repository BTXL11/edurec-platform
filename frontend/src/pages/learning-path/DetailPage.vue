<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { completePathStep, getLearningPath, getPathResources } from '@/api/learning'
import ResourceCard from '@/components/resource/ResourceCard.vue'
import type { LearningPath, PathStep, Resource } from '@/types'

const route = useRoute()
const pathId = computed(() => Number(route.params.id))

const path = ref<LearningPath | null>(null)
const resources = ref<Resource[]>([])
const currentPointName = ref('')
const loading = ref(true)
const completing = ref(false)
const error = ref('')

const doneCount = computed(
  () => path.value?.steps.filter((step) => step.status === 'done').length ?? 0,
)
const progress = computed(() => {
  const total = path.value?.steps.length ?? 0
  return total === 0 ? 0 : Math.round((doneCount.value / total) * 100)
})

const statusLabels: Record<PathStep['status'], string> = {
  done: '已完成',
  current: '当前学习',
  locked: '未解锁',
}
const statusTypes: Record<PathStep['status'], 'success' | 'primary' | 'info'> = {
  done: 'success',
  current: 'primary',
  locked: 'info',
}

async function loadResources() {
  if (!path.value || path.value.status === 'completed') {
    resources.value = []
    currentPointName.value = ''
    return
  }
  try {
    const data = await getPathResources(pathId.value, 9)
    resources.value = data.list
    currentPointName.value = data.knowledge_point?.name ?? ''
  } catch {
    // 资源加载失败不阻断路径展示
    resources.value = []
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    path.value = await getLearningPath(pathId.value)
    await loadResources()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
    path.value = null
  } finally {
    loading.value = false
  }
}

async function completeCurrent() {
  if (!path.value) return
  completing.value = true
  try {
    path.value = await completePathStep(pathId.value, path.value.current_index)
    await loadResources()
    ElMessage.success(
      path.value.status === 'completed' ? '整条学习路径已完成' : '已完成本节，下一节已解锁',
    )
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  } finally {
    completing.value = false
  }
}

onMounted(load)
watch(pathId, load)
</script>

<template>
  <div class="mx-auto max-w-5xl px-6 py-10">
    <RouterLink :to="{ name: 'knowledge' }" class="text-sm text-primary hover:underline">
      ← 返回知识点导航
    </RouterLink>

    <div v-if="loading" class="mt-8">
      <el-skeleton :rows="8" animated />
    </div>
    <el-empty v-else-if="error" :description="error" class="mt-16" />

    <template v-else-if="path">
      <header class="mt-4 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 class="text-2xl font-bold text-ink">{{ path.target_name }} · 学习路径</h1>
          <p class="mt-2 text-sm text-ink-secondary">
            按先修关系逐节解锁；已掌握的内容会跳过，只推荐路径上还没掌握的部分。
          </p>
        </div>
        <el-tag :type="path.status === 'completed' ? 'success' : 'primary'">
          {{ path.status === 'completed' ? '已全部完成' : '进行中' }}
        </el-tag>
      </header>

      <div class="mt-6">
        <el-progress :percentage="progress" :stroke-width="10" />
        <p class="mt-1 text-xs text-ink-muted">
          已完成 {{ doneCount }} / {{ path.steps.length }} 节
        </p>
      </div>

      <ol class="mt-6 space-y-3">
        <li
          v-for="(step, index) in path.steps"
          :key="step.code + index"
          class="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-surface px-4 py-3"
          :class="step.status === 'current' ? 'border-primary shadow-sm' : 'border-border'"
        >
          <div class="flex items-center gap-3">
            <span
              class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
              :class="step.status === 'done'
                ? 'bg-emerald-100 text-emerald-700'
                : step.status === 'current'
                  ? 'bg-primary text-white'
                  : 'bg-bg text-ink-muted'"
            >
              {{ step.status === 'done' ? '✓' : index + 1 }}
            </span>
            <div>
              <span
                class="text-sm font-medium"
                :class="step.status === 'locked' ? 'text-ink-muted' : 'text-ink'"
              >
                {{ step.name }}
              </span>
              <span class="ml-2 text-xs text-ink-muted">第 {{ step.level }} 层</span>
            </div>
          </div>
          <div class="flex items-center gap-3 text-xs text-ink-muted">
            <span>掌握度 {{ Math.round(step.mastery * 100) }}%</span>
            <span>资源 {{ step.resource_count }}</span>
            <el-tag size="small" :type="statusTypes[step.status]">
              {{ statusLabels[step.status] }}
            </el-tag>
          </div>
        </li>
      </ol>

      <el-alert
        v-if="path.status === 'completed'"
        class="mt-6"
        type="success"
        :closable="false"
        title="这条学习路径已经学完"
        description="可以回到知识点导航，选择一个更深入的目标继续生成路径。"
      />

      <template v-else>
        <section class="mt-8">
          <div class="flex items-end justify-between">
            <h2 class="text-lg font-semibold text-ink">
              当前一节{{ currentPointName ? `：${currentPointName}` : '' }}
            </h2>
            <el-button type="primary" round :loading="completing" @click="completeCurrent">
              学完本节，解锁下一节
            </el-button>
          </div>

          <div
            v-if="resources.length > 0"
            class="mt-4 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3"
          >
            <ResourceCard v-for="resource in resources" :key="resource.id" :resource="resource" />
          </div>
          <el-empty
            v-else
            description="本节暂无课程资源，可先到搜索页按关键词采集，或直接完成本节"
            class="mt-4"
          />
        </section>
      </template>
    </template>
  </div>
</template>
