import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { KnowledgeDetail, Resource } from '@/types'
import { getKnowledgePoint } from '@/api/knowledge'
import KnowledgeDetailPage from '../DetailPage.vue'

vi.mock('vue-router', async () => {
  const { reactive } = await import('vue')
  const route = reactive({ params: { code: 'mathematical-analysis' } })
  return {
    route,
    useRoute: () => route,
    useRouter: () => ({ push: vi.fn() }),
    RouterLink: { template: '<a><slot /></a>' },
  }
})
vi.mock('@/api/knowledge', () => ({ getKnowledgePoint: vi.fn() }))

const mockedDetail = vi.mocked(getKnowledgePoint)

const resource: Resource = {
  id: 63, title: '高等数学全程教学', description: '宋浩老师', cover_url: null,
  type: 'course', category: { id: 2, name: 'B站视频' }, tags: ['高数'], metadata: {},
  author: '宋浩', source_url: 'https://www.bilibili.com/video/BV1xx',
  avg_rating: 4.8, view_count: 1000,
  created_at: '2026-07-01T08:00:00Z', updated_at: '2026-07-15T10:00:00Z',
}

const detail: KnowledgeDetail = {
  point: {
    id: 2, code: 'mathematical-analysis', name: '数学分析', description: '极限与积分的严格理论',
    level: 2, resource_count: 26, concept_count: 0,
  },
  nodes: [
    { id: 1, code: 'calculus', name: '高等数学', description: '', level: 1, resource_count: 43, concept_count: 91 },
    { id: 2, code: 'mathematical-analysis', name: '数学分析', description: '', level: 2, resource_count: 26, concept_count: 0 },
  ],
  edges: [{ from: 1, to: 2 }],
  resources: [resource],
  resource_total: 1,
  concepts: [],
  concept_edges: [],
}

async function mountPage() {
  const wrapper = mount(KnowledgeDetailPage, { global: { plugins: [ElementPlus] } })
  await flushPromises()
  return wrapper
}

describe('KnowledgeDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedDetail.mockResolvedValue(detail)
  })

  it('展示学科信息、先修图与课程资源', async () => {
    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('数学分析')
    expect(wrapper.text()).toContain('第 2 层')
    expect(wrapper.text()).toContain('课程资源 26')
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.text()).toContain('高等数学')
    expect(wrapper.text()).toContain('高等数学全程教学')
    expect(mockedDetail).toHaveBeenCalledWith('mathematical-analysis', 12)
  })

  it('概念层为空时给出说明', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('暂未匹配到概念层')
  })

  it('有概念时展示概念标签与数量', async () => {
    mockedDetail.mockResolvedValue({
      ...detail,
      point: { ...detail.point, concept_count: 2 },
      concepts: [
        { id: 10, code: 'K_极限_数学', name: '极限', field: '数学', point_code: 'mathematical-analysis', level: 1, explanation: '数列极限' },
        { id: 11, code: 'K_级数_数学', name: '级数', field: '数学', point_code: 'mathematical-analysis', level: 2, explanation: '级数收敛' },
      ],
      concept_edges: [{ from: 10, to: 11 }],
    })

    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('极限')
    expect(wrapper.text()).toContain('级数')
    expect(wrapper.text()).toContain('共 2 个概念')
    expect(wrapper.text()).toContain('概念先修边 1 条')
  })
})
