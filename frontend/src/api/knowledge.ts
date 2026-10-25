import { get } from './client'
import type { KnowledgeDetail, KnowledgeNode, Page, Resource } from '@/types'

/** 知识点（学科级）列表 */
export function listKnowledgePoints() {
  return get<KnowledgeNode[]>('/knowledge-points')
}

/** 知识点详情：学科先修图 + 概念子图 + 资源 */
export function getKnowledgePoint(code: string, limit = 12) {
  return get<KnowledgeDetail>(`/knowledge-points/${encodeURIComponent(code)}`, {
    params: { limit },
  })
}

/** 知识点下的资源分页 */
export function listKnowledgeResources(code: string, page = 1, pageSize = 12) {
  return get<Page<Resource> & { knowledge_point: KnowledgeNode }>(
    `/knowledge-points/${encodeURIComponent(code)}/resources`,
    { params: { page, page_size: pageSize } },
  )
}
