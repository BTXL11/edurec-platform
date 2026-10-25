<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getLearningProfile, submitQuestionnaire } from '@/api/learning'
import type { LearningProfileItem } from '@/types'

const router = useRouter()

const items = ref<LearningProfileItem[]>([])
const answers = ref<Record<string, number>>({})
const loading = ref(true)
const saving = ref(false)
const error = ref('')

const levelLabels = ['没接触过', '了解概念', '学过并练习', '熟练']
const levelOptions = levelLabels.map((label, value) => ({ label, value }))

const grouped = computed(() => {
  const map = new Map<number, LearningProfileItem[]>()
  for (const item of items.value) {
    const level = Math.max(1, item.level || 1)
    const group = map.get(level) ?? []
    group.push(item)
    map.set(level, group)
  }
  return [...map.entries()]
    .sort(([a], [b]) => a - b)
    .map(([level, list]) => ({ level, items: list }))
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = await getLearningProfile()
    const initial: Record<string, number> = {}
    for (const item of items.value) {
      initial[item.point_code] = Math.min(3, Math.max(0, Math.round(item.mastery * 3)))
    }
    answers.value = initial
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await submitQuestionnaire(answers.value)
    ElMessage.success('已保存，学习路径会按你的水平跳过已掌握内容')
    router.push({ name: 'knowledge' })
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-4xl px-6 py-10">
    <div class="flex items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold">初始学习问卷</h1>
        <p class="mt-2 text-sm text-ink-secondary">
          按你的真实水平自评每个知识点，系统会据此跳过已掌握内容，只推荐路径上还没掌握的部分。
          以后可以随时回来重做。
        </p>
      </div>
      <RouterLink :to="{ name: 'knowledge' }" class="shrink-0 text-sm text-primary hover:underline">
        先去看知识点 →
      </RouterLink>
    </div>

    <div v-if="loading" class="mt-8">
      <el-skeleton :rows="8" animated />
    </div>
    <el-empty v-else-if="error" :description="error" class="mt-16" />

    <template v-else>
      <section v-for="group in grouped" :key="group.level" class="mt-8">
        <h2 class="text-lg font-semibold text-ink">第 {{ group.level }} 层</h2>
        <div class="mt-3 space-y-3">
          <div
            v-for="item in group.items"
            :key="item.point_code"
            class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border bg-surface px-4 py-3"
          >
            <div class="min-w-40">
              <span class="text-sm font-medium text-ink">{{ item.name }}</span>
              <span v-if="item.source" class="ml-2 text-xs text-ink-muted">
                （当前：{{ item.source === 'behavior' ? '行为估计' : '已填写' }}）
              </span>
            </div>
            <el-radio-group v-model="answers[item.point_code]" size="small">
              <el-radio-button
                v-for="option in levelOptions"
                :key="option.value"
                :value="option.value"
              >
                {{ option.label }}
              </el-radio-button>
            </el-radio-group>
          </div>
        </div>
      </section>

      <div class="mt-8 flex items-center gap-3">
        <el-button type="primary" round :loading="saving" @click="save">保存并返回知识点</el-button>
        <span class="text-xs text-ink-muted">0 = 没接触，3 = 熟练；掌握度 ≥ 80% 的节点会从路径中跳过</span>
      </div>
    </template>
  </div>
</template>
