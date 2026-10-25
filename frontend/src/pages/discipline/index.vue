<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listDisciplines } from '@/api/discipline'
import type { Discipline } from '@/types'

const items = ref<Discipline[]>([])
const loading = ref(true)
const error = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = await listDisciplines()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-7xl px-6 py-10">
    <div>
      <h1 class="text-2xl font-bold">学科导航</h1>
      <p class="mt-2 text-sm text-ink-secondary">
        先选学科，再进入学科页查看知识点与课程资源；没有资源的学科会自动去 B 站抓取。
      </p>
    </div>

    <div v-if="loading" class="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <el-skeleton v-for="i in 9" :key="i" :rows="2" animated />
    </div>
    <div v-else-if="error" class="py-24 text-center text-sm text-red-500">{{ error }}</div>
    <el-empty v-else-if="items.length === 0" description="暂无学科" class="mt-8" />

    <div v-else class="mt-8 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
      <RouterLink
        v-for="item in items"
        :key="item.code"
        :to="{ name: 'discipline', params: { code: item.code } }"
        class="group rounded-lg border border-border bg-surface p-5 transition hover:border-primary hover:shadow-md"
      >
        <div class="flex items-start justify-between gap-3">
          <h2 class="text-lg font-semibold text-ink group-hover:text-primary">{{ item.name }}</h2>
          <el-tag v-if="item.resource_count === 0" size="small" type="warning">自动抓取</el-tag>
        </div>
        <p class="mt-2 line-clamp-2 text-sm text-ink-secondary">{{ item.description }}</p>
        <div class="mt-4 flex items-center gap-4 text-xs text-ink-muted">
          <span>知识点 {{ item.point_count }}</span>
          <span>资源 {{ item.resource_count }}</span>
        </div>
      </RouterLink>
    </div>
  </div>
</template>
