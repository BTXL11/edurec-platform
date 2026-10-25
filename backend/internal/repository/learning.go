package repository

import (
	"fmt"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"gorm.io/gorm"
)

// LearningRepository 学习画像与学习路径的数据访问接口
type LearningRepository interface {
	UpsertProfile(profile *model.LearnerProfile) error
	ListProfiles(userID uint) ([]model.LearnerProfile, error)
	SaveQuestionnaire(userID uint, answers string) error

	CreatePath(path *model.LearningPath) error
	UpdatePath(path *model.LearningPath) error
	FindPathByID(id uint) (*model.LearningPath, error)
	FindActivePath(userID uint, targetCode string) (*model.LearningPath, error)
	ArchiveActivePaths(userID uint, targetCode string) error

	// UserEngagementByPoint 统计用户在各知识点下的行为次数，用于行为侧的掌握度估计
	UserEngagementByPoint(userID uint) (map[uint]int64, error)
}

type LearningRepo struct {
	db *gorm.DB
}

func NewLearningRepository(db *gorm.DB) *LearningRepo {
	return &LearningRepo{db: db}
}

// UpsertProfile 同一用户同一知识点只保留一行，掌握度取较高值（问卷自评不会被一次低行为覆盖）
func (r *LearningRepo) UpsertProfile(profile *model.LearnerProfile) error {
	var existing model.LearnerProfile
	err := r.db.Where("user_id = ? AND point_code = ?", profile.UserID, profile.PointCode).
		First(&existing).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		if err := r.db.Create(profile).Error; err != nil {
			return fmt.Errorf("创建学习画像失败: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("查询学习画像失败: %w", err)
	}

	if profile.Mastery > existing.Mastery {
		if err := r.db.Model(&existing).Updates(map[string]any{
			"mastery": profile.Mastery,
			"source":  profile.Source,
		}).Error; err != nil {
			return fmt.Errorf("更新学习画像失败: %w", err)
		}
	}
	return nil
}

func (r *LearningRepo) ListProfiles(userID uint) ([]model.LearnerProfile, error) {
	var items []model.LearnerProfile
	if err := r.db.Where("user_id = ?", userID).
		Order("point_code ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("查询学习画像失败: %w", err)
	}
	return items, nil
}

func (r *LearningRepo) SaveQuestionnaire(userID uint, answers string) error {
	response := &model.QuestionnaireResponse{UserID: userID, Answers: answers}
	if err := r.db.Create(response).Error; err != nil {
		return fmt.Errorf("保存问卷失败: %w", err)
	}
	return nil
}

func (r *LearningRepo) CreatePath(path *model.LearningPath) error {
	if err := r.db.Create(path).Error; err != nil {
		return fmt.Errorf("创建学习路径失败: %w", err)
	}
	return nil
}

func (r *LearningRepo) UpdatePath(path *model.LearningPath) error {
	if err := r.db.Save(path).Error; err != nil {
		return fmt.Errorf("更新学习路径失败: %w", err)
	}
	return nil
}

func (r *LearningRepo) FindPathByID(id uint) (*model.LearningPath, error) {
	var path model.LearningPath
	err := r.db.First(&path, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询学习路径失败: %w", err)
	}
	return &path, nil
}

func (r *LearningRepo) FindActivePath(userID uint, targetCode string) (*model.LearningPath, error) {
	var path model.LearningPath
	err := r.db.Where("user_id = ? AND target_code = ? AND status = ?", userID, targetCode, "active").
		Order("id DESC").First(&path).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询学习路径失败: %w", err)
	}
	return &path, nil
}

// ArchiveActivePaths 新建路径前把同一目标的旧活动路径归档，避免出现多条并行的「当前路径」
func (r *LearningRepo) ArchiveActivePaths(userID uint, targetCode string) error {
	if err := r.db.Model(&model.LearningPath{}).
		Where("user_id = ? AND target_code = ? AND status = ?", userID, targetCode, "active").
		Update("status", "archived").Error; err != nil {
		return fmt.Errorf("归档旧学习路径失败: %w", err)
	}
	return nil
}

func (r *LearningRepo) UserEngagementByPoint(userID uint) (map[uint]int64, error) {
	type row struct {
		PointID uint
		Total   int64
	}
	var rows []row
	err := r.db.Raw(`
		SELECT rkp.point_id AS point_id, COUNT(*) AS total
		FROM user_behaviors b
		JOIN resource_knowledge_points rkp
		  ON rkp.resource_id = b.resource_id AND rkp.deleted_at IS NULL
		WHERE b.user_id = ? AND b.deleted_at IS NULL
		GROUP BY rkp.point_id
	`, userID).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("统计用户行为分布失败: %w", err)
	}
	out := make(map[uint]int64, len(rows))
	for _, item := range rows {
		out[item.PointID] = item.Total
	}
	return out, nil
}
