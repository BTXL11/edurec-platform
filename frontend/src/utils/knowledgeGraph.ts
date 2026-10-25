import type { KnowledgeEdge, KnowledgeNode } from '@/types'

export interface KnowledgeGraphLayoutOptions {
  /** 列间距：同一层级（level）一列 */
  dx?: number
  /** 行间距：同层节点纵向排列 */
  dy?: number
  nodeWidth?: number
  nodeHeight?: number
  padding?: number
}

export interface PositionedKnowledgeNode extends KnowledgeNode {
  x: number
  y: number
}

export interface PositionedKnowledgeEdge {
  from: PositionedKnowledgeNode
  to: PositionedKnowledgeNode
}

export interface KnowledgeGraphLayout {
  nodes: PositionedKnowledgeNode[]
  edges: PositionedKnowledgeEdge[]
  width: number
  height: number
  /** 最大层级，便于调用方渲染「第 N 层」标注 */
  levels: number
}

/**
 * 把知识点先修图排成「层级从左到右、同层上下排列」的布局。
 *
 * 不引入图形库：知识点规模小（十几个学科 / 几十个概念），
 * 用 level 直接决定横坐标即可得到稳定、可读的 DAG 视图。
 * 同层节点按 id 排序，并在该列内纵向居中，避免长列把整图拉偏。
 */
export function layoutKnowledgeGraph(
  nodes: KnowledgeNode[],
  edges: KnowledgeEdge[],
  options: KnowledgeGraphLayoutOptions = {},
): KnowledgeGraphLayout {
  const dx = options.dx ?? 260
  const dy = options.dy ?? 100
  const nodeWidth = options.nodeWidth ?? 180
  const nodeHeight = options.nodeHeight ?? 56
  const padding = options.padding ?? 24

  if (nodes.length === 0) {
    return { nodes: [], edges: [], width: 0, height: 0, levels: 0 }
  }

  const byLevel = new Map<number, KnowledgeNode[]>()
  for (const node of nodes) {
    const level = Math.max(1, node.level || 1)
    const group = byLevel.get(level) ?? []
    group.push(node)
    byLevel.set(level, group)
  }

  const levels = Math.max(...byLevel.keys())
  const maxRows = Math.max(...[...byLevel.values()].map((group) => group.length))

  const positioned: PositionedKnowledgeNode[] = []
  for (const [level, group] of byLevel) {
    const sorted = [...group].sort((a, b) => a.id - b.id)
    const offset = (maxRows - sorted.length) / 2
    sorted.forEach((node, index) => {
      positioned.push({
        ...node,
        x: padding + (level - 1) * dx,
        y: padding + (offset + index) * dy,
      })
    })
  }

  const byId = new Map(positioned.map((node) => [node.id, node]))
  const positionedEdges: PositionedKnowledgeEdge[] = []
  for (const edge of edges) {
    const from = byId.get(edge.from)
    const to = byId.get(edge.to)
    if (!from || !to) continue
    positionedEdges.push({ from, to })
  }

  return {
    nodes: positioned,
    edges: positionedEdges,
    width: padding * 2 + (levels - 1) * dx + nodeWidth,
    height: padding * 2 + (maxRows - 1) * dy + nodeHeight,
    levels,
  }
}
