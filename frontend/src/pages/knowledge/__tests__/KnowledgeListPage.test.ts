import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { KnowledgeNode } from '@/types'
import { listKnowledgePoints } from '@/api/knowledge'
import KnowledgeListPage from '../index.vue'

vi.mock('@/api/knowledge', () => ({ listKnowledgePoints: vi.fn() }))

const mockedList = vi.mocked(listKnowledgePoints)

const points: KnowledgeNode[] = [
  {
    id: 1, code: 'linear-algebra', name: '线性代数', description: '向量空间与矩阵',
    level: 1, resource_count: 15, concept_count: 19,
  },
  {
    id: 2, code: 'analysis', name: '数学分析', description: '极限与积分的严格理论',
    level: 2, resource_count: 26, concept_count: 0,
  },
]

async function mountPage() {
  const wrapper = mount(KnowledgeListPage, {
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return wrapper
}

describe('KnowledgeListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedList.mockResolvedValue(points)
  })

  it('按层级分组展示知识点与资源/概念数', async () => {
    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('知识点导航')
    expect(wrapper.text()).toContain('第 1 层')
    expect(wrapper.text()).toContain('线性代数')
    expect(wrapper.text()).toContain('第 2 层')
    expect(wrapper.text()).toContain('数学分析')
    expect(wrapper.text()).toContain('课程资源 15')
    expect(wrapper.text()).toContain('概念 19')
    expect(wrapper.findAllComponents(RouterLinkStub).length).toBeGreaterThanOrEqual(2)
  })

  it('关键词过滤按名称或描述生效', async () => {
    const wrapper = await mountPage()
    const input = wrapper.find('input[placeholder^="搜索知识点"]')
    await input.setValue('线性')
    await flushPromises()

    expect(wrapper.text()).toContain('线性代数')
    expect(wrapper.text()).not.toContain('数学分析')
  })

  it('接口失败时展示错误', async () => {
    mockedList.mockRejectedValue(new Error('网络错误'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('网络错误')
  })
})
