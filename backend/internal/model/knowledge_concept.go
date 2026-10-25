package model

import "gorm.io/gorm"

// KnowledgeConcept 概念级知识点（来自 EMBEDD-ER 的 MOOCCube 概念图）。
//
// 与 KnowledgePoint（学科级入口）是两层关系：一个学科下挂若干概念，
// 概念之间由先修边构成更细的图。PointCode 为空表示尚未归入任何学科。
//
// Code 直接用概念 ID（形如 K_随机变量_数学），保证跨库可追溯；
// Field 取自 concept-field 数据（数学 / 计算机科学技术）。
type KnowledgeConcept struct {
	gorm.Model
	Code        string `gorm:"uniqueIndex;size:191;not null" json:"code"`
	Name        string `gorm:"size:191;not null;index" json:"name"`
	Field       string `gorm:"size:64;index" json:"field"`
	PointCode   string `gorm:"size:64;index" json:"point_code"`
	Level       int    `gorm:"default:1;index" json:"level"`
	Explanation string `gorm:"type:text" json:"explanation"`
}

// KnowledgeConceptEdge 概念先修边：From 是 To 的先修
type KnowledgeConceptEdge struct {
	gorm.Model
	FromID uint `gorm:"not null;uniqueIndex:idx_kc_edge" json:"from_id"`
	ToID   uint `gorm:"not null;uniqueIndex:idx_kc_edge" json:"to_id"`
}
