package model

import "gorm.io/gorm"

// KnowledgePoint 知识点：把散落的资源聚合成「数学分析」「机器学习」这样的学习节点。
//
// Code 是对外稳定标识（如 linear-algebra），URL 与前端路由都用它，避免依赖自增 ID。
// Level 表示在有向无环图里的层级（1 = 无先修），由先修边推导后写入，便于列表与图谱分层展示。
type KnowledgePoint struct {
	gorm.Model
	Code        string `gorm:"uniqueIndex;size:64;not null" json:"code"`
	Name        string `gorm:"size:128;not null;index" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Level       int    `gorm:"default:1;index" json:"level"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`
}

// KnowledgePointEdge 知识点先修关系：From 是 To 的先修（From 学完才能学 To）。
// (from_id, to_id) 唯一，种子与手工维护都按这一对幂等写入。
type KnowledgePointEdge struct {
	gorm.Model
	FromID uint `gorm:"not null;uniqueIndex:idx_kp_edge" json:"from_id"`
	ToID   uint `gorm:"not null;uniqueIndex:idx_kp_edge" json:"to_id"`
}

// ResourceKnowledgePoint 资源与知识点的多对多关联。
// 一个视频可以同时属于「线性代数」和「机器学习」；Weight 表示匹配强度（0-1），
// 用于在同一知识点内给资源排序，也便于后续按可信度过滤。
type ResourceKnowledgePoint struct {
	gorm.Model
	ResourceID uint    `gorm:"not null;uniqueIndex:idx_rkp" json:"resource_id"`
	PointID    uint    `gorm:"not null;uniqueIndex:idx_rkp;index" json:"point_id"`
	Weight     float32 `gorm:"default:1" json:"weight"`
}
