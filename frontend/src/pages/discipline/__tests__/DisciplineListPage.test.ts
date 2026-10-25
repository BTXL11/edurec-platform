import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { Discipline } from '@/types'
import { listDisciplines } from '@/api/discipline'
import DisciplineListPage from '../index.vue'

vi.mock('@/api/discipline', () => ({ listDisciplines: vi.fn() }))

const mockedList = vi.mocked(listDisciplines)

const disciplines: Discipline[] = [
  { code: 'math', name: '数学', description: '分析、代数、几何', point_count: 12, resource_count: 225 },
  { code: 'chemistry', name: '化学', description: '无机、有机', point_count: 4, resource_count: 0 },
]

async function mountPage() {
  const wrapper = mount(DisciplineListPage, {
    global: { plugins: [ElementPlus], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return wrapper
}

describe('DisciplineListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedList.mockResolvedValue(disciplines)
  })

  it('展示全部学科及其知识点/资源数', async () => {
    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('学科导航')
    expect(wrapper.text()).toContain('数学')
    expect(wrapper.text()).toContain('化学')
    expect(wrapper.text()).toContain('知识点 12')
    expect(wrapper.text()).toContain('资源 225')
    expect(wrapper.findAllComponents(RouterLinkStub).length).toBeGreaterThanOrEqual(2)
  })

  it('没有资源的学科标记为自动抓取', async () => {
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('自动抓取')
  })

  it('接口失败时展示错误', async () => {
    mockedList.mockRejectedValue(new Error('网络错误'))
    const wrapper = await mountPage()
    expect(wrapper.text()).toContain('网络错误')
  })
})
