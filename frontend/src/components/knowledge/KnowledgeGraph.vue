<script setup lang="ts">
import { computed } from 'vue'
import type { KnowledgeEdge, KnowledgeNode } from '@/types'
import { layoutKnowledgeGraph } from '@/utils/knowledgeGraph'

const props = withDefaults(
  defineProps<{
    nodes: KnowledgeNode[]
    edges: KnowledgeEdge[]
    currentId?: number
    compact?: boolean
  }>(),
  { currentId: 0, compact: false },
)

const emit = defineEmits<{ (e: 'select', node: KnowledgeNode): void }>()

const layout = computed(() =>
  layoutKnowledgeGraph(props.nodes, props.edges, {
    dx: props.compact ? 220 : 260,
    dy: props.compact ? 78 : 100,
    nodeWidth: props.compact ? 150 : 180,
    nodeHeight: props.compact ? 48 : 56,
  }),
)

const nodeWidth = computed(() => (props.compact ? 150 : 180))
const nodeHeight = computed(() => (props.compact ? 48 : 56))
const arrowId = `kg-arrow-${Math.random().toString(36).slice(2, 8)}`

function edgePath(fromX: number, fromY: number, toX: number, toY: number): string {
  const startX = fromX + nodeWidth.value
  const midX = (startX + toX) / 2
  return `M ${startX} ${fromY} C ${midX} ${fromY}, ${midX} ${toY}, ${toX} ${toY}`
}

function labelOf(node: KnowledgeNode): string {
  return node.name.length > 11 ? `${node.name.slice(0, 11)}…` : node.name
}

function onSelect(node: KnowledgeNode) {
  emit('select', node)
}
</script>

<template>
  <div class="overflow-x-auto">
    <svg
      v-if="layout.nodes.length > 0"
      :viewBox="`0 0 ${layout.width} ${layout.height}`"
      :width="layout.width"
      :height="layout.height"
      role="img"
      aria-label="知识点先修关系图"
      class="min-w-full"
    >
      <defs>
        <marker
          :id="arrowId"
          markerWidth="8"
          markerHeight="8"
          refX="7"
          refY="4"
          orient="auto"
        >
          <path d="M 0 0 L 8 4 L 0 8 z" fill="#94a3b8" />
        </marker>
      </defs>

      <path
        v-for="(edge, index) in layout.edges"
        :key="index"
        :d="edgePath(edge.from.x, edge.from.y + nodeHeight / 2, edge.to.x, edge.to.y + nodeHeight / 2)"
        fill="none"
        stroke="#cbd5e1"
        stroke-width="1.5"
        :marker-end="`url(#${arrowId})`"
      />

      <g
        v-for="node in layout.nodes"
        :key="node.id"
        :transform="`translate(${node.x}, ${node.y})`"
        role="button"
        tabindex="0"
        class="cursor-pointer focus:outline-none"
        @click="onSelect(node)"
        @keydown.enter="onSelect(node)"
        @keydown.space.prevent="onSelect(node)"
      >
        <rect
          :width="nodeWidth"
          :height="nodeHeight"
          rx="10"
          :fill="node.id === currentId ? '#3b82f6' : '#ffffff'"
          :stroke="node.id === currentId ? '#2563eb' : '#cbd5e1'"
          stroke-width="1.5"
        />
        <text
          :x="nodeWidth / 2"
          :y="compact ? 20 : 23"
          text-anchor="middle"
          font-size="13"
          font-weight="600"
          :fill="node.id === currentId ? '#ffffff' : '#1f2937'"
        >
          {{ labelOf(node) }}
        </text>
        <text
          :x="nodeWidth / 2"
          :y="compact ? 37 : 42"
          text-anchor="middle"
          font-size="11"
          :fill="node.id === currentId ? '#dbeafe' : '#6b7280'"
        >
          资源 {{ node.resource_count }} · 概念 {{ node.concept_count }}
        </text>
      </g>
    </svg>
  </div>
</template>
