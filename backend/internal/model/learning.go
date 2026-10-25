package model

import "gorm.io/gorm"

// MasteryThreshold 掌握度阈值：达到即视为该知识点已掌握，不必写入学习路径。
const MasteryThreshold = 0.8

// LearnerProfile 学习画像：用户对某个知识点的掌握度（0-1）。
//
// 来源有三类：初始问卷（自评）、日常行为（观看/收藏等）、完成学习路径中的该节。
// (user_id, point_code) 唯一，同一用户同一知识点只保留一行，取较高的掌握度。
type LearnerProfile struct {
	gorm.Model
	UserID    uint    `gorm:"not null;uniqueIndex:idx_profile_user_point" json:"user_id"`
	PointCode string  `gorm:"size:64;not null;uniqueIndex:idx_profile_user_point" json:"point_code"`
	Mastery   float32 `gorm:"type:decimal(3,2);default:0" json:"mastery"`
	Source    string  `gorm:"size:32" json:"source"` // questionnaire / behavior / complete
}

// QuestionnaireResponse 初始问卷的一次提交记录（保留历史，便于分析与重做）
type QuestionnaireResponse struct {
	gorm.Model
	UserID  uint   `gorm:"index" json:"user_id"`
	Answers string `gorm:"type:json" json:"answers"` // {"linear-algebra": 0-3, ...}
}

// LearningPath 用户针对某个目标知识点的学习路径。
// Steps 是 JSON 数组（[]PathStep），按先修顺序排列；CurrentIndex 指向当前应学的一节。
type LearningPath struct {
	gorm.Model
	UserID       uint   `gorm:"not null;index:idx_lpath_user_target" json:"user_id"`
	TargetCode   string `gorm:"size:64;not null;index:idx_lpath_user_target" json:"target_code"`
	Steps        string `gorm:"type:json" json:"steps"`
	CurrentIndex int    `gorm:"default:0" json:"current_index"`
	Status       string `gorm:"size:16;default:'active'" json:"status"` // active / completed
}

// PathStep 学习路径中的一节。
type PathStep struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	Level         int     `json:"level"`
	Mastery       float32 `json:"mastery"`
	Status        string  `json:"status"` // done / current / locked
	ResourceCount int64   `json:"resource_count"`
}
