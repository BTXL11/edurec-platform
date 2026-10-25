import { get, post } from './client'
import type { LearningPath, LearningProfileItem, Page, Resource } from '@/types'

/** 当前用户的知识画像（全部知识点的掌握度） */
export function getLearningProfile() {
  return get<LearningProfileItem[]>('/learning-profile')
}

/** 提交初始问卷：知识点 code -> 自评等级 0-3 */
export function submitQuestionnaire(answers: Record<string, number>) {
  return post<LearningProfileItem[]>('/learning-profile/questionnaire', { answers })
}

/** 为某个目标知识点生成学习路径 */
export function createLearningPath(target: string) {
  return post<LearningPath>('/learning-paths', { target })
}

export function getLearningPath(id: number) {
  return get<LearningPath>(`/learning-paths/${id}`)
}

/** 完成路径中的某一节，成功后返回更新后的路径 */
export function completePathStep(id: number, index: number) {
  return post<LearningPath>(`/learning-paths/${id}/complete`, { index })
}

/** 当前步骤对应的课程资源 */
export function getPathResources(id: number, limit = 12) {
  return get<Page<Resource> & { knowledge_point: { id: number; code: string; name: string } }>(
    `/learning-paths/${id}/resources`,
    { params: { limit } },
  )
}
