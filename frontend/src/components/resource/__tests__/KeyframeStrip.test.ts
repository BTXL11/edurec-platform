import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import KeyframeStrip from '../KeyframeStrip.vue'

const sprites = [
  'https://bimp.hdslb.com/videoshot/1-0001.jpg',
  'https://bimp.hdslb.com/videoshot/1-0002.jpg',
]

function mountStrip(props: Record<string, unknown> = {}) {
  return mount(KeyframeStrip, {
    props: {
      frames: { sprites, x_len: 10, y_len: 10, x_size: 480, y_size: 270, count: 200 },
      ...props,
    },
  })
}

describe('KeyframeStrip', () => {
  it('按网格参数裁切雪碧图并默认渲染 8 帧', () => {
    const wrapper = mountStrip()
    expect(wrapper.text()).toContain('关键帧')
    expect(wrapper.text()).toContain('共 200 帧')

    const cells = wrapper.findAll('div[title^="关键帧："]')
    expect(cells).toHaveLength(8)

    const style = cells[0].attributes('style') ?? ''
    expect(style).toContain('background-image')
    expect(style).toContain('1-0001.jpg')
    // 10 × 10 网格：background-size 应放大到 1000%
    expect(style).toContain('background-size: 1000% 1000%')
    expect(style).toContain('background-position: 0% 0%')
  })

  it('max 控制展示帧数', () => {
    const wrapper = mountStrip({ max: 3 })
    expect(wrapper.findAll('div[title^="关键帧："]')).toHaveLength(3)
  })

  it('没有雪碧图时不渲染任何内容', () => {
    const wrapper = mountStrip({
      frames: { sprites: [], x_len: 10, y_len: 10, count: 0 },
    })
    expect(wrapper.findAll('div[title^="关键帧："]')).toHaveLength(0)
    expect(wrapper.text()).not.toContain('关键帧')
  })
})
