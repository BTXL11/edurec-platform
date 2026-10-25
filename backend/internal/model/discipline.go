package model

import "gorm.io/gorm"

// Discipline 学科：主页面的一级入口（数学、物理、化学、计算机……）。
//
// 知识点（KnowledgePoint）通过 DisciplineCode 挂在学科下；
// Keywords 是 JSON 数组，既用于把已有资源匹配到学科，也作为无资源时自动爬取的搜索词。
type Discipline struct {
	gorm.Model
	Code        string `gorm:"uniqueIndex;size:64;not null" json:"code"`
	Name        string `gorm:"size:64;not null;index" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Keywords    string `gorm:"type:json" json:"keywords"`
	SortOrder   int    `gorm:"default:0" json:"sort_order"`
}
