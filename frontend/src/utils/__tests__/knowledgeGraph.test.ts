import { describe, expect, it } from 'vitest'
import type { KnowledgeEdge, KnowledgeNode } from '@/types'
import { layoutKnowledgeGraph } from '@/utils/knowledgeGraph'

function node(id: number, level: number, name = `n${id}`): KnowledgeNode {
  return {
    id,
    code: `code-${id}`,
    name,
    description: '',
    level,
    resource_count: id,
    concept_count: 0,
  }
}

describe('layoutKnowledgeGraph', () => {
  it('按层级从左到右排列，同层纵向居中', () => {
    const nodes = [node(1, 1), node(2, 1), node(3, 2)]
    const edges: KnowledgeEdge[] = [
      { from: 1, to: 3 },
      { from: 2, to: 3 },
    ]

    const layout = layoutKnowledgeGraph(nodes, edges, {
      dx: 200, dy: 80, padding: 10, nodeWidth: 100, nodeHeight: 40,
    })

    const byId = new Map(layout.nodes.map((item) => [item.id, item]))
    expect(byId.get(1)!.x).toBe(10)
    expect(byId.get(2)!.x).toBe(10)
    expect(byId.get(3)!.x).toBe(210)
    // 第一层两个节点 y=10/90；第二层单个节点居中在 50
    expect(byId.get(1)!.y).toBe(10)
    expect(byId.get(2)!.y).toBe(90)
    expect(byId.get(3)!.y).toBe(50)

    expect(layout.levels).toBe(2)
    expect(layout.width).toBe(10 * 2 + 200 + 100)
    expect(layout.height).toBe(10 * 2 + 80 + 40)
    expect(layout.edges).toHaveLength(2)
  })

  it('忽略指向不存在节点的边', () => {
    const layout = layoutKnowledgeGraph([node(1, 1)], [{ from: 1, to: 99 }])
    expect(layout.edges).toHaveLength(0)
  })

  it('空图返回零尺寸', () => {
    expect(layoutKnowledgeGraph([], [])).toEqual({
      nodes: [], edges: [], width: 0, height: 0, levels: 0,
    })
  })
})
